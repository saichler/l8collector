package service

import (
	"github.com/saichler/l8collector/go/collector/common"
	"github.com/saichler/l8parser/go/parser/boot"
	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/types"
	"github.com/saichler/l8srlz/go/serialize/object"
)

func (this *HostCollector) preBoot(job *types.CJob, poll *types.Poll) bool {
	if poll.What == "ipaddress" {
		obj := object.NewEncode()
		for _, h := range this.device.Hosts {
			for _, c := range h.Configs {
				obj.Add(c.Addr)
				job.Result = obj.Data()
				break
			}
			break
		}
		this.ipDiscovered = true
		if this.ipDiscovered && this.stateDiscovered {
			this.boot()
		}
		return true
	} else if poll.What == "devicestatus" {
		obj := object.NewEncode()
		protocolState := make(map[int32]bool)
		this.collectors.Iterate(func(k, v interface{}) {
			key := k.(types.Protocol)
			p := v.(common.ProtocolCollector)
			protocolState[int32(key)] = p.Online()
		})
		obj.Add(protocolState)
		job.Result = obj.Data()
		this.stateDiscovered = true
		if this.ipDiscovered && this.stateDiscovered {
			this.boot()
		}
		return true
	}
	return false
}

func (this *HostCollector) boot() {
	if this.loadedBoot {
		return
	}
	bootPollList, err := pollaris.PollarisByGroup(this.service.vnic.Resources(), common.BOOT_GROUP,
		"", "", "", "", "", "")
	if err != nil {
		this.service.vnic.Resources().Logger().Error("Failed to boot: ", err.Error())
		return
	}
	for _, pollName := range bootPollList {
		err := this.jobsQueue.InsertJob(pollName.Name, "", "", "", "", "", "", 0, 0)
		if err != nil {
			this.service.vnic.Resources().Logger().Error(err)
		}
	}
	this.loadedBoot = true
}

func (this *HostCollector) bootDetailDevice(job *types.CJob) {
	if this.loadedDeviceSpecific {
		return
	}
	if job.Result == nil || len(job.Result) < 5 {
		this.service.vnic.Resources().Logger().Error("HostCollector.loadPolls:", job.JobName, " ", "Has empty Result")
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
		if plrs.Name != "mib2" {
			this.loadedDeviceSpecific = true
			this.jobsQueue.InsertJob(plrs.Name, "", "", "", "", "", "", 0, 0)
		}
	}
}
