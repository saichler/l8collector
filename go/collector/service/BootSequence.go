package service

import (
	"time"

	"github.com/saichler/collect/go/types"
	"github.com/saichler/l8collector/go/collector/common"
	"github.com/saichler/l8parser/go/parser/boot"
	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/types/l8poll"
	"github.com/saichler/l8srlz/go/serialize/object"
)

type BootState struct {
	jobNames map[string]bool
	stage    int
}

func (this *HostCollector) newBootState(stage int) *BootState {
	bs := &BootState{}
	bs.stage = stage
	bs.jobNames = make(map[string]bool)
	pollList, err := pollaris.PollarisByGroup(this.service.vnic.Resources(), common.BootStages[stage],
		"", "", "", "", "", "")
	if err != nil {
		this.service.vnic.Resources().Logger().Error("Boot stage ", stage, " does not exist,skipping")
		return bs
	}
	for _, pollrs := range pollList {
		hasProtocol := false
		for _, poll := range pollrs.Polling {
			_, ok := this.collectors.Get(poll.Protocol)
			if ok {
				bs.jobNames[poll.Name] = false
				hasProtocol = true
			}
		}
		if hasProtocol {
			err = this.jobsQueue.InsertJob(pollrs.Name, "", "", "", "", "", "", 0, 0)
			if err != nil {
				this.service.vnic.Resources().Logger().Error("Error adding pollaris to boot: ", err)
			}
		}
	}
	return bs
}

func (this *BootState) isComplete() bool {
	for _, complete := range this.jobNames {
		if !complete {
			return false
		}
	}
	return true
}

func (this *BootState) doStaticJob(job *l8poll.CJob, hostColletor *HostCollector) bool {
	sjob, ok := staticJobs[job.JobName]
	if ok {
		sjob.do(job, hostColletor)
		_, ok = this.jobNames[job.JobName]
		if ok {
			this.jobNames[job.JobName] = true
		}
		return true
	}
	return false
}

func (this *BootState) jobComplete(job *l8poll.CJob) {
	_, ok := this.jobNames[job.JobName]
	if ok {
		this.jobNames[job.JobName] = true
	}
}

func (this *HostCollector) bootDetailDevice(job *l8poll.CJob) {
	if this.detailDeviceLoaded {
		return
	}
	if job.Result == nil || len(job.Result) < 3 {
		this.service.vnic.Resources().Logger().Error("HostCollector.loadPolls: ", job.TargetId, " has sysmib empty Result")
		return
	}
	enc := object.NewDecode(job.Result, 0, this.service.vnic.Resources().Registry())
	data, err := enc.Get()
	if err != nil {
		this.service.vnic.Resources().Logger().Error("HostCollector, loadPolls: ", job.TargetId, " has sysmib error ", err.Error())
		return
	}
	cmap, ok := data.(*types.CMap)
	if !ok {
		this.service.vnic.Resources().Logger().Error("HostCollector, loadPolls: ", job.TargetId, " systemMib not A CMap")
		return
	}
	strData, ok := cmap.Data[".1.3.6.1.2.1.1.2.0"]
	if !ok {
		this.service.vnic.Resources().Logger().Error("HostCollector, loadPolls: ", job.TargetId, " sysmib does not contain sysoid")
		return
	}

	enc = object.NewDecode(strData, 0, this.service.vnic.Resources().Registry())
	byteInterface, _ := enc.Get()
	sysoid, _ := byteInterface.(string)
	this.service.vnic.Resources().Logger().Info("HostCollector, loadPolls: ", job.TargetId, " discovered sysoid =", sysoid)
	if sysoid == "" {
		this.service.vnic.Resources().Logger().Error("HostCollector, loadPolls: ", job.TargetId, " - sysoid is blank!")
		/* when there is DebugEnabled
		for k, v := range cmap.Data {
			enc = object.NewDecode(v, 0, this.service.vnic.Resources().Registry())
			val, _ := enc.Get()
			this.service.vnic.Resources().Logger().Debug("Key =", k, " value=", val)
		}*/
		return
	}

	plrs := boot.GetPollarisByOid(sysoid)
	plc := pollaris.Pollaris(this.service.vnic.Resources())
	plc.Add(plrs, false)
	if plrs != nil {
		if plrs.Name != "boot03" {
			this.service.vnic.Resources().Logger().Info("HostCollector, loadPolls: ", job.TargetId, " discovered pollaris by sysoid ", plrs.Name, " by systoid:", sysoid)
			this.detailDeviceLoaded = true
			go this.insertCustomJobs(plrs.Name)
		}
	}
}

func (this *HostCollector) insertCustomJobs(pollarisName string) {
	time.Sleep(time.Second * 300)
	this.jobsQueue.InsertJob(pollarisName, "", "", "", "", "", "", 0, 0)
}
