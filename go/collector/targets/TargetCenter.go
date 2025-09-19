package targets

import (
	"github.com/saichler/l8pollaris/go/types/l8poll"
	"github.com/saichler/l8services/go/services/dcache"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8reflect/go/reflect/introspecting"
)

type TargetCenter struct {
	devices ifs.IDistributedCache
}

func newDeviceCenter(serviceName string, serviceArea byte, resources ifs.IResources, listener ifs.IServiceCacheListener) *TargetCenter {
	this := &TargetCenter{}
	node, _ := resources.Introspector().Inspect(&l8poll.L8C_Target{})
	introspecting.AddPrimaryKeyDecorator(node, "TargetId")
	this.devices = dcache.NewDistributedCache(serviceName, serviceArea, &l8poll.L8C_Target{}, nil,
		listener, resources)
	return this
}

func (this *TargetCenter) Shutdown() {
	this.devices = nil
}

func (this *TargetCenter) Post(device *l8poll.L8C_Target, isNotification bool) bool {
	elem, _ := this.devices.Get(device)
	this.devices.Post(device, isNotification)
	return elem != nil
}

func (this *TargetCenter) Put(device *l8poll.L8C_Target, isNotification bool) bool {
	elem, _ := this.devices.Get(device)
	this.devices.Put(device, isNotification)
	return elem != nil
}

func (this *TargetCenter) Patch(device *l8poll.L8C_Target, isNotification bool) bool {
	elem, _ := this.devices.Get(device)
	this.devices.Patch(device, isNotification)
	return elem != nil
}

func (this *TargetCenter) Delete(device *l8poll.L8C_Target, isNotification bool) bool {
	elem, _ := this.devices.Get(device)
	this.devices.Delete(device, isNotification)
	return elem != nil
}

func (this *TargetCenter) DeviceById(id string) *l8poll.L8C_Target {
	filter := &l8poll.L8C_Target{TargetId: id}
	d, _ := this.devices.Get(filter)
	device, _ := d.(*l8poll.L8C_Target)
	return device
}

func (this *TargetCenter) HostConnection(deviceId, hostId string) map[int32]*l8poll.L8T_Connection {
	if this == nil {
		panic("nil")
	}
	filter := &l8poll.L8C_Target{TargetId: deviceId}
	d, _ := this.devices.Get(filter)
	device, _ := d.(*l8poll.L8C_Target)
	if device == nil {
		return nil
	}
	return device.Hosts[hostId].Configs
}

func Configs(resource ifs.IResources, serviceArea byte) *TargetCenter {
	sp, ok := resource.Services().ServiceHandler(ServiceName, serviceArea)
	if !ok {
		return nil
	}
	return (sp.(*TargetService)).configCenter
}
