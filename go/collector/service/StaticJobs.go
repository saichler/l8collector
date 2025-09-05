package service

import (
	"github.com/saichler/l8collector/go/collector/common"
	"github.com/saichler/l8pollaris/go/types"
	"github.com/saichler/l8srlz/go/serialize/object"
)

var staticJobs = map[string]StaticJob{(&IpAddressJob{}).what(): &IpAddressJob{}, (&DeviceStatusJob{}).what(): &DeviceStatusJob{}}

type StaticJob interface {
	what() string
	do(job *types.CJob, hostCollector *HostCollector)
}

type IpAddressJob struct{}

func (this *IpAddressJob) what() string {
	return "ipaddress"
}

func (this *IpAddressJob) do(job *types.CJob, hostCollector *HostCollector) {
	obj := object.NewEncode()
	for _, h := range hostCollector.device.Hosts {
		for _, c := range h.Configs {
			obj.Add(c.Addr)
			job.Result = obj.Data()
			break
		}
		break
	}
}

type DeviceStatusJob struct{}

func (this *DeviceStatusJob) what() string {
	return "devicestatus"
}

func (this *DeviceStatusJob) do(job *types.CJob, hostCollector *HostCollector) {
	obj := object.NewEncode()
	protocolState := make(map[int32]bool)
	hostCollector.collectors.Iterate(func(k, v interface{}) {
		key := k.(types.Protocol)
		p := v.(common.ProtocolCollector)
		protocolState[int32(key)] = p.Online()
	})
	obj.Add(protocolState)
	job.Result = obj.Data()
}
