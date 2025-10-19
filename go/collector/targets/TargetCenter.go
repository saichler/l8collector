package targets

import (
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8reflect/go/reflect/introspecting"
	"github.com/saichler/l8services/go/services/dcache"
	"github.com/saichler/l8types/go/ifs"
)

type TargetCenter struct {
	devices ifs.IDistributedCache
}

func newDeviceCenter(sla *ifs.ServiceLevelAgreement, vnic ifs.IVNic) *TargetCenter {
	this := &TargetCenter{}
	node, _ := vnic.Resources().Introspector().Inspect(&l8tpollaris.L8PTarget{})
	introspecting.AddPrimaryKeyDecorator(node, "TargetId")
	this.devices = dcache.NewDistributedCache(sla.ServiceName(), sla.ServiceArea(), &l8tpollaris.L8PTarget{}, nil,
		vnic, vnic.Resources())
	return this
}

func (this *TargetCenter) Shutdown() {
	this.devices = nil
}

func (this *TargetCenter) Post(device *l8tpollaris.L8PTarget, isNotification bool) bool {
	elem, _ := this.devices.Get(device)
	this.devices.Post(device, isNotification)
	return elem != nil
}

func (this *TargetCenter) Put(device *l8tpollaris.L8PTarget, isNotification bool) bool {
	elem, _ := this.devices.Get(device)
	this.devices.Put(device, isNotification)
	return elem != nil
}

func (this *TargetCenter) Patch(device *l8tpollaris.L8PTarget, isNotification bool) bool {
	elem, _ := this.devices.Get(device)
	this.devices.Patch(device, isNotification)
	return elem != nil
}

func (this *TargetCenter) Delete(device *l8tpollaris.L8PTarget, isNotification bool) bool {
	elem, _ := this.devices.Get(device)
	this.devices.Delete(device, isNotification)
	return elem != nil
}

func (this *TargetCenter) DeviceById(id string) *l8tpollaris.L8PTarget {
	filter := &l8tpollaris.L8PTarget{TargetId: id}
	d, _ := this.devices.Get(filter)
	device, _ := d.(*l8tpollaris.L8PTarget)
	return device
}

func (this *TargetCenter) HostConnection(deviceId, hostId string) map[int32]*l8tpollaris.L8PHostProtocol {
	if this == nil {
		panic("nil")
	}
	filter := &l8tpollaris.L8PTarget{TargetId: deviceId}
	d, _ := this.devices.Get(filter)
	device, _ := d.(*l8tpollaris.L8PTarget)
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
