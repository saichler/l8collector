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
)

type HostCollector struct {
	service              *CollectorService
	device               *types.Device
	hostId               string
	collectors           *maps.SyncMap
	jobsQueue            *JobsQueue
	running              bool
	loadedDeviceSpecific bool
	loadedBoot           bool
	ipDiscovered         bool
	stateDiscovered      bool
}

func newHostCollector(device *types.Device, hostId string, service *CollectorService) *HostCollector {
	hc := &HostCollector{}
	hc.device = device
	hc.hostId = hostId
	hc.collectors = maps.NewSyncMap()
	hc.service = service
	hc.jobsQueue = NewJobsQueue(device.DeviceId, hostId, service, device.InventoryService, device.ParsingService)
	hc.running = true
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

	bootPollList, err := pollaris.PollarisByGroup(this.service.vnic.Resources(), common.PRE_BOOT_GROUP,
		"", "", "", "", "", "")
	if err != nil {
		return err
	}
	for _, pollName := range bootPollList {
		err := this.jobsQueue.InsertJob(pollName.Name, "", "", "", "", "", "", 0, 0)
		if err != nil {
			this.service.vnic.Resources().Logger().Error(err)
		}
	}

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

	bootPollaris, err := pollaris.PollarisByGroup(this.service.vnic.Resources(), common.PRE_BOOT_GROUP,
		"", "", "", "", "", "")
	if err != nil {
		return err
	}
	for _, pr := range bootPollaris {
		this.jobsQueue.InsertJob(pr.Name, "", "", "", "", "", "", 0, 0)
	}

	go this.collect()

	return nil
}

func (this *HostCollector) collect() {
	if common.SmoothForSimulators {
		//Sleep up to 5 minutes before starting to collect
		//so the collection for all devices will be smoother on the simulator
		time.Sleep(time.Second * time.Duration(common.RandomSecondWithin5Minutes()))
	}
	this.service.vnic.Resources().Logger().Info("** Starting Collection on host ", this.hostId)
	pc := pollaris.Pollaris(this.service.vnic.Resources())
	var jobFailCount int
	var job *types.CJob
	var waitTime int64
	for this.running {
		if job == nil || jobFailCount == 0 {
			job, waitTime = this.jobsQueue.Pop()
			if job != nil {
				this.service.vnic.Resources().Logger().Info("Poped job ", job.PollarisName, ":", job.JobName)
			} else {
				this.service.vnic.Resources().Logger().Info("No Job, waitTime ", waitTime)
			}
		}
		if job != nil {
			poll := pc.Poll(job.PollarisName, job.JobName)
			if poll == nil {
				this.service.vnic.Resources().Logger().Error("cannot find poll for device id ", this.device.DeviceId)
				continue
			}
			MarkStart(job)

			if this.preBoot(job, poll) {
				MarkEnded(job)
				this.jobComplete(job)
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
			}
			if job.Error != "" {
				jobFailCount++
				time.Sleep(time.Second * 5)
			} else {
				jobFailCount = 0
				job = nil
			}
			if jobFailCount == 3 {
				this.jobsQueue.DisableJob(job)
				jobFailCount = 0
				job = nil
			}
		} else {
			this.service.vnic.Resources().Logger().Info("No more jobs, next job in ", waitTime, " seconds.")
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
		return nil, errors.New("Unknown Protocol " + config.Protocol.String())
	}
	err := protocolCollector.Init(config, resource)
	return protocolCollector, err
}

func (this *HostCollector) jobComplete(job *types.CJob) {
	err := this.service.vnic.RoundRobin(job.PService.ServiceName, byte(job.PService.ServiceArea), ifs.POST, job)
	if err != nil {
		this.service.vnic.Resources().Logger().Error("HostCollector:", err.Error())
	}
	if job.JobName == "systemMib" {
		this.service.vnic.Resources().Logger().Info("SystemMib job result")
		this.bootDetailDevice(job)
	}
}
