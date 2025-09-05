package service

import (
	"github.com/saichler/l8collector/go/collector/common"
	"github.com/saichler/l8parser/go/parser/boot"
	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/types"
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
		for _, poll := range pollrs.Polling {
			bs.jobNames[poll.Name] = false
		}
		err = this.jobsQueue.InsertJob(pollrs.Name, "", "", "", "", "", "", 0, 0)
		if err != nil {
			this.service.vnic.Resources().Logger().Error("Error adding pollaris to boot: ", err)
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

func (this *BootState) doStaticJob(job *types.CJob, hostColletor *HostCollector) bool {
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

func (this *BootState) jobComplete(job *types.CJob) {
	_, ok := this.jobNames[job.JobName]
	if ok {
		this.jobNames[job.JobName] = true
	}
}

func (this *HostCollector) bootDetailDevice(job *types.CJob) {
	if this.detailDeviceLoaded {
		return
	}
	if job.Result == nil || len(job.Result) < 5 {
		this.service.vnic.Resources().Logger().Error("HostCollector.loadPolls:", job.DeviceId, " ", job.JobName, " ", "Has empty Result")
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
	plc := pollaris.Pollaris(this.service.vnic.Resources())
	plc.Add(plrs, false)
	if plrs != nil {
		this.service.vnic.Resources().Logger().Info("HostCollector, loadPolls: found pollaris by sysoid ", plrs.Name, " by systoid:", sysoid)
		if plrs.Name != "boot02" {
			this.detailDeviceLoaded = true
			this.jobsQueue.InsertJob(plrs.Name, "", "", "", "", "", "", 0, 0)
		}
	}
}
