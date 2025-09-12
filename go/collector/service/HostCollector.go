package service

import (
	"errors"
	"time"

	"github.com/saichler/l8collector/go/collector/common"
	"github.com/saichler/l8collector/go/collector/protocols/k8s"
	"github.com/saichler/l8collector/go/collector/protocols/snmp"
	"github.com/saichler/l8collector/go/collector/protocols/ssh"
	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/types"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/maps"
	"github.com/saichler/l8utils/go/utils/strings"
)

type HostCollector struct {
	service            *CollectorService
	device             *types.Device
	hostId             string
	collectors         *maps.SyncMap
	jobsQueue          *JobsQueue
	running            bool
	currentBootStage   int
	bootStages         []*BootState
	detailDeviceLoaded bool
}

func newHostCollector(device *types.Device, hostId string, service *CollectorService) *HostCollector {
	hc := &HostCollector{}
	hc.device = device
	hc.hostId = hostId
	hc.collectors = maps.NewSyncMap()
	hc.service = service
	hc.jobsQueue = NewJobsQueue(device.DeviceId, hostId, service, device.InventoryService, device.ParsingService)
	hc.running = true
	hc.bootStages = make([]*BootState, 5)
	hc.service.vnic.RegisterServiceBatch(device.ParsingService.ServiceName, byte(device.ParsingService.ServiceArea), ifs.M_Proximity, 5)
	return hc
}

func (this *HostCollector) update() error {
	host := this.device.Hosts[this.hostId]
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
}

func (this *HostCollector) start() error {
	host := this.device.Hosts[this.hostId]
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
	this.service.vnic.Resources().Logger().Info("** Starting Collection on host ", this.hostId)
	pc := pollaris.Pollaris(this.service.vnic.Resources())
	var job *types.CJob
	var waitTime int64
	for this.running {

		job, waitTime = this.jobsQueue.Pop()
		if job != nil {
			this.service.vnic.Resources().Logger().Debug("Poped job ", job.PollarisName, ":", job.JobName)
		} else {
			this.service.vnic.Resources().Logger().Debug("No Job, waitTime ", waitTime)
		}

		if job != nil {
			poll := pc.Poll(job.PollarisName, job.JobName)
			if poll == nil {
				this.service.vnic.Resources().Logger().Error(strings.New("cannot find poll ", job.PollarisName, " - ", job.JobName, " for device id ").String(), this.device.DeviceId)
				continue
			}
			MarkStart(job)

			if this.bootStages[0].doStaticJob(job, this) {
				MarkEnded(job)
				this.jobComplete(job)
				if this.bootStages[0].isComplete() && this.bootStages[1] == nil {
					this.bootStages[1] = this.newBootState(1)
					this.currentBootStage = 1
					if common.SmoothForSimulators {
						//Sleep up to 5 minutes before starting to collect
						//so the collection for all devices will be smoother on the simulator
						time.Sleep(time.Second * time.Duration(common.RandomSecondWithin5Minutes()))
					}
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
				this.service.vnic.Resources().Logger().Error("Job ", job.DeviceId, " - ", job.PollarisName, " - ",
					job.JobName, " has failed ", job.ErrorCount, " in a row, disabling job")
				this.jobsQueue.DisableJob(job)
			}
		} else {
			this.service.vnic.Resources().Logger().Debug("No more jobs, next job in ", waitTime, " seconds.")
			time.Sleep(time.Second * time.Duration(waitTime))
		}
	}
	this.service.vnic.Resources().Logger().Info("Host collection for device ", this.device.DeviceId, " host ", this.hostId, " has ended.")
	this.service = nil
}

func (this *HostCollector) execJob(job *types.CJob) bool {
	pc := pollaris.Pollaris(this.service.vnic.Resources())
	poll := pc.Poll(job.PollarisName, job.JobName)
	if poll == nil {
		this.service.vnic.Resources().Logger().Error("cannot find poll for device id ", this.device.DeviceId)
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

func newProtocolCollector(config *types.Connection, resource ifs.IResources) (common.ProtocolCollector, error) {
	var protocolCollector common.ProtocolCollector
	if config.Protocol == types.Protocol_PSSH {
		protocolCollector = &ssh.SshCollector{}
	} else if config.Protocol == types.Protocol_PSNMPV2 {
		protocolCollector = &snmp.SNMPv2Collector{}
	} else if config.Protocol == types.Protocol_PK8s {
		protocolCollector = &k8s.Kubernetes{}
	} else {
		return nil, errors.New(strings.New("Unknown Protocol ", config.Protocol.String()).String())
	}
	err := protocolCollector.Init(config, resource)
	return protocolCollector, err
}

func (this *HostCollector) jobComplete(job *types.CJob) {
	if !jobHasChange(job) {
		this.service.vnic.Resources().Logger().Info("Job", job.JobName, " has no change")
		if job.Error != "" {
			this.service.vnic.Resources().Logger().Error("Job ", job.DeviceId, " - ", job.PollarisName,
				" - ", job.JobName, " has an error:", job.Error)
		}
		return
	}
	err := this.service.vnic.Proximity(job.PService.ServiceName, byte(job.PService.ServiceArea), ifs.POST, job)
	if err != nil {
		this.service.vnic.Resources().Logger().Error("HostCollector:", err.Error())
	}
	if job.JobName == "systemMib" {
		this.service.vnic.Resources().Logger().Debug("SystemMib for ", job.DeviceId, " was received")
		this.bootDetailDevice(job)
	}
}

func jobHasChange(job *types.CJob) bool {
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
