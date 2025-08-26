package service

import (
	"errors"
	"time"

	"github.com/saichler/l8collector/go/collector/common"
	"github.com/saichler/l8collector/go/collector/protocols/k8s"
	"github.com/saichler/l8collector/go/collector/protocols/snmp"
	"github.com/saichler/l8collector/go/collector/protocols/ssh"
	"github.com/saichler/l8parser/go/parser/boot"
	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/types"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/maps"
)

type HostCollector struct {
	service    *CollectorService
	device     *types.Device
	hostId     string
	collectors *maps.SyncMap
	jobsQueue  *JobsQueue
	running    bool
	loaded     bool
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

	bootPollList, err := pollaris.PollarisByGroup(this.service.vnic.Resources(), common.BOOT_GROUP,
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

	bootPollaris, err := pollaris.PollarisByGroup(this.service.vnic.Resources(), common.BOOT_GROUP,
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
	this.service.vnic.Resources().Logger().Info("** Starting Collection on host ", this.hostId)
	pc := pollaris.Pollaris(this.service.vnic.Resources())
	for this.running {
		job, waitTime := this.jobsQueue.Pop()
		if job != nil {
			this.service.vnic.Resources().Logger().Info("Poped job ", job.PollarisName, ":", job.JobName)
		} else {
			this.service.vnic.Resources().Logger().Info("No Job, waitTime ", waitTime)
		}
		if job != nil {
			poll := pc.Poll(job.PollarisName, job.JobName)
			if poll == nil {
				this.service.vnic.Resources().Logger().Error("cannot find poll for device id ", this.device.DeviceId)
				continue
			}
			MarkStart(job)
			c, ok := this.collectors.Get(poll.Protocol)
			if !ok {
				MarkEnded(job)
				continue
			}
			c.(common.ProtocolCollector).Exec(job)
			MarkEnded(job)
			if this.running {
				this.jobComplete(job)
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
	err := this.service.vnic.Proximity(job.PService.ServiceName, byte(job.PService.ServiceArea), ifs.POST, job)
	if err != nil {
		this.service.vnic.Resources().Logger().Error("HostCollector:", err.Error())
	}
	if job.JobName == "systemMib" {
		this.service.vnic.Resources().Logger().Info("SystemMib job result")
		this.loadPolls(job)
	}
}

func (this *HostCollector) loadPolls(job *types.CJob) {
	if this.loaded {
		return
	}
	enc := object.NewDecode(job.Result, 0, this.service.vnic.Resources().Registry())
	data, err := enc.Get()
	if err != nil {
		this.service.vnic.Resources().Logger().Error("HostCollector, loadPolls:", err.Error())
		return
	}
	cmap, ok := data.(*types.CMap)
	if !ok {
		this.service.vnic.Resources().Logger().Error("HostCollector, loadPolls: systemMib not A CMap")
		return
	}
	strData, ok := cmap.Data[".1.3.6.1.2.1.1.2.0"]
	if !ok {
		this.service.vnic.Resources().Logger().Error("HostCollector, loadPolls: cannot find sysoid")
		return
	}

	enc = object.NewDecode(strData, 0, this.service.vnic.Resources().Registry())
	byteInterface, _ := enc.Get()
	sysoidBytes, ok := byteInterface.([]byte)
	sysoid := string(sysoidBytes)
	this.service.vnic.Resources().Logger().Info("HostCollector, loadPolls, sysoid =", sysoid)
	if sysoid == "" {
		this.service.vnic.Resources().Logger().Error("HostCollector, loadPolls: sysoid is blank ")
		for k, v := range cmap.Data {
			enc = object.NewDecode(v, 0, this.service.vnic.Resources().Registry())
			val, _ := enc.Get()
			this.service.vnic.Resources().Logger().Info("Key =", k, " value=", val)
		}
	}

	plrs := boot.GetPollarisByOid(sysoid)
	if plrs != nil {
		this.service.vnic.Resources().Logger().Info("HostCollector, loadPolls: found pollaris by sysoid ", plrs.Name, " by systoid:", sysoid)
		this.loaded = true
		if plrs.Name != "mib2" {
			this.jobsQueue.InsertJob(plrs.Name, "", "", "", "", "", "", 0, 0)
		}
	}
}
