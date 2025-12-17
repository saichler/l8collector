package service

import (
	"errors"
	"github.com/saichler/l8types/go/types/l8services"
	"time"

	"github.com/saichler/l8collector/go/collector/common"
	"github.com/saichler/l8collector/go/collector/protocols/graphql"
	"github.com/saichler/l8collector/go/collector/protocols/k8s"
	"github.com/saichler/l8collector/go/collector/protocols/rest"
	"github.com/saichler/l8collector/go/collector/protocols/snmp"
	"github.com/saichler/l8collector/go/collector/protocols/ssh"
	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/pollaris/targets"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/maps"
	"github.com/saichler/l8utils/go/utils/strings"
)

type HostCollector struct {
	service          *CollectorService
	target           *l8tpollaris.L8PTarget
	hostId           string
	collectors       *maps.SyncMap
	jobsQueue        *JobsQueue
	running          bool
	currentBootStage int
	bootStages       []*BootState
	pollarisName     string
}

func newHostCollector(target *l8tpollaris.L8PTarget, hostId string, service *CollectorService) *HostCollector {
	hc := &HostCollector{}
	hc.target = target
	hc.hostId = hostId
	hc.collectors = maps.NewSyncMap()
	hc.service = service
	hc.jobsQueue = NewJobsQueue(target, hostId, service)
	hc.running = true
	hc.bootStages = make([]*BootState, 5)
	ps, pa := targets.Links.Parser(target.LinksId)
	if ps != "" {
		parserLink := &l8services.L8ServiceLink{ZsideServiceName: ps, ZsideServiceArea: int32(pa),
			Mode: int32(ifs.M_Proximity), Interval: 5}
		hc.service.vnic.RegisterServiceLink(parserLink)
	}
	return hc
}

func (this *HostCollector) update() error {
	host := this.target.Hosts[this.hostId]
	for _, config := range host.Configs {
		exist := this.collectors.Contains(config.Protocol)
		if !exist {
			col, err := newProtocolCollector(config, this.service.vnic.Resources())
			if err != nil {
				return this.service.vnic.Resources().Logger().Error(err)
			}
			if col != nil {
				this.collectors.Put(config.Protocol, col)
			}
		}
	}

	this.bootStages = make([]*BootState, 5)
	this.bootStages[0] = this.newBootState(0)

	return nil
}

func (this *HostCollector) stop() {
	this.running = false
	this.collectors.Iterate(func(k, v interface{}) {
		c := v.(common.ProtocolCollector)
		c.Disconnect()
	})
	this.collectors = nil
	this.jobsQueue.Shutdown()
	this.jobsQueue = nil
	this.bootStages = nil
	this.target = nil
	this.service = nil
}

func (this *HostCollector) start() error {
	host := this.target.Hosts[this.hostId]
	for _, config := range host.Configs {
		col, err := newProtocolCollector(config, this.service.vnic.Resources())
		if err != nil {
			this.service.vnic.Resources().Logger().Error(err)
		}
		if col != nil {
			this.collectors.Put(config.Protocol, col)
		}
	}

	this.bootStages[0] = this.newBootState(0)

	go this.collect()

	return nil
}

func (this *HostCollector) collect() {
	// Capture references before they may be cleared by stop()
	resources := this.service.vnic.Resources()
	targetId := this.target.TargetId
	hostId := this.hostId

	pc := pollaris.Pollaris(resources)
	var job *l8tpollaris.CJob
	var waitTime int64
	for this.running {

		job, waitTime = this.jobsQueue.Pop()
		if job != nil {
			resources.Logger().Debug("Poped job ", job.PollarisName, ":", job.JobName)
		} else {
			resources.Logger().Debug("No Job, waitTime ", waitTime)
		}

		if job != nil {
			poll := pc.Poll(job.PollarisName, job.JobName)
			if poll == nil {
				resources.Logger().Error(strings.New("cannot find poll ", job.PollarisName, " - ", job.JobName, " for device id ").String(), targetId)
				continue
			}
			MarkStart(job)

			if this.currentBootStage < len(this.bootStages) && this.bootStages[this.currentBootStage].doStaticJob(job, this) {
				MarkEnded(job)
				this.jobComplete(job)
				if this.bootStages[this.currentBootStage].isComplete() && this.currentBootStage < len(this.bootStages)-1 {
					this.currentBootStage++
					this.bootStages[this.currentBootStage] = this.newBootState(this.currentBootStage)
				}
				continue
			}

			c, ok := this.collectors.Get(poll.Protocol)
			if !ok {
				MarkEnded(job)
				this.jobsQueue.DisableJob(job)
				continue
			}

			c.(common.ProtocolCollector).Exec(job)
			MarkEnded(job)
			if this.running {
				this.jobComplete(job)
				if this.currentBootStage < len(this.bootStages) {
					this.bootStages[this.currentBootStage].jobComplete(job)
					for this.bootStages[this.currentBootStage].isComplete() {
						this.currentBootStage++
						if this.currentBootStage >= len(this.bootStages) {
							break
						}
						this.bootStages[this.currentBootStage] = this.newBootState(this.currentBootStage)
					}
				}
			}

			if job.ErrorCount >= 5 {
				resources.Logger().Error("Job ", job.TargetId, " - ", job.PollarisName, " - ",
					job.JobName, " has failed ", job.ErrorCount, " in a row.")
			}
		} else {
			resources.Logger().Debug("No more jobs, next job in ", waitTime, " seconds.")
			time.Sleep(time.Second * time.Duration(waitTime))
		}
	}
	resources.Logger().Info("Host collection for device ", targetId, " host ", hostId, " has ended.")
}

func (this *HostCollector) execJob(job *l8tpollaris.CJob) bool {
	pc := pollaris.Pollaris(this.service.vnic.Resources())
	poll := pc.Poll(job.PollarisName, job.JobName)
	if poll == nil {
		panic(this.target.TargetId + ": cannot find poll " + job.PollarisName + "/" + job.JobName)
		this.service.vnic.Resources().Logger().Error("cannot find poll for device id ", this.target.TargetId)
		return false
	}
	MarkStart(job)
	c, ok := this.collectors.Get(poll.Protocol)
	if !ok {
		MarkEnded(job)
		return false
	}
	c.(common.ProtocolCollector).Exec(job)
	MarkEnded(job)
	return true
}

func newProtocolCollector(config *l8tpollaris.L8PHostProtocol, resource ifs.IResources) (common.ProtocolCollector, error) {
	var protocolCollector common.ProtocolCollector
	if config.Protocol == l8tpollaris.L8PProtocol_L8PGraphQL {
		protocolCollector = &graphql.GraphQlCollector{}
	} else if config.Protocol == l8tpollaris.L8PProtocol_L8PRESTCONF {
		protocolCollector = &rest.RestCollector{}
	} else if config.Protocol == l8tpollaris.L8PProtocol_L8PSSH {
		protocolCollector = &ssh.SshCollector{}
	} else if config.Protocol == l8tpollaris.L8PProtocol_L8PPSNMPV2 {
		protocolCollector = &snmp.SNMPv2Collector{}
	} else if config.Protocol == l8tpollaris.L8PProtocol_L8PKubectl {
		protocolCollector = &k8s.Kubernetes{}
	} else {
		return nil, errors.New(strings.New("Unknown Protocol ", config.Protocol.String()).String())
	}
	err := protocolCollector.Init(config, resource)
	return protocolCollector, err
}

func (this *HostCollector) jobComplete(job *l8tpollaris.CJob) {
	if job.Error != "" {
		this.service.vnic.Resources().Logger().Error("Job ", job.TargetId, " - ", job.PollarisName,
			" - ", job.JobName, " has an error:", job.Error)
		job.Cadence.Current = 0
		return
	}

	if !jobHasChange(job) {
		this.service.vnic.Resources().Logger().Debug("Job", job.JobName, " has no change")
		return
	}

	pService, pArea := targets.Links.Parser(job.LinksId)

	err := this.service.vnic.Proximity(pService, pArea, ifs.POST, job)
	if err != nil {
		this.service.vnic.Resources().Logger().Error("HostCollector:", err.Error())
	}
	if job.JobName == "systemMib" {
		this.service.vnic.Resources().Logger().Debug("SystemMib for ", job.TargetId, " was received")
		this.bootDetailDevice(job)
	}
}

func jobHasChange(job *l8tpollaris.CJob) bool {
	if job.Always {
		return true
	}
	if job.Result != nil && job.Cadence.Current < int32(len(job.Cadence.Cadences)-1) {
		job.Cadence.Current++
	}
	if job.LastResult == nil && job.Result == nil {
		return false
	} else if job.LastResult == nil && job.Result != nil {
		return true
	} else if job.Result == nil {
		return true
	}
	if len(job.Result) != len(job.LastResult) {
		return true
	}
	for i := 0; i < len(job.Result); i++ {
		if job.Result[i] != job.LastResult[i] {
			return true
		}
	}
	return false
}
