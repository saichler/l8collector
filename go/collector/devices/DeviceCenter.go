package devices

import (
	"github.com/saichler/l8pollaris/go/types"
	"github.com/saichler/l8services/go/services/dcache"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/reflect/go/reflect/introspecting"
)

type DeviceCenter struct {
	devices ifs.IDistributedCache
}

func newDeviceCenter(serviceName string, serviceArea byte, resources ifs.IResources, listener ifs.IServiceCacheListener) *DeviceCenter {
	this := &DeviceCenter{}
	node, _ := resources.Introspector().Inspect(&types.Device{})
	introspecting.AddPrimaryKeyDecorator(node, "DeviceId")
	this.devices = dcache.NewDistributedCache(serviceName, serviceArea, &types.Device{}, nil,
		listener, resources)
	return this
}

func (this *DeviceCenter) Shutdown() {
	this.devices = nil
}

func (this *DeviceCenter) Post(device *types.Device, isNotification bool) bool {
	elem, _ := this.devices.Get(device)
	this.devices.Post(device, isNotification)
	return elem != nil
}

func (this *DeviceCenter) Put(device *types.Device, isNotification bool) bool {
	elem, _ := this.devices.Get(device)
	this.devices.Put(device, isNotification)
	return elem != nil
}

func (this *DeviceCenter) Patch(device *types.Device, isNotification bool) bool {
	elem, _ := this.devices.Get(device)
	this.devices.Patch(device, isNotification)
	return elem != nil
}

func (this *DeviceCenter) Delete(device *types.Device, isNotification bool) bool {
	elem, _ := this.devices.Get(device)
	this.devices.Delete(device, isNotification)
	return elem != nil
}

func (this *DeviceCenter) DeviceById(id string) *types.Device {
	filter := &types.Device{DeviceId: id}
	d, _ := this.devices.Get(filter)
	device, _ := d.(*types.Device)
	return device
}

func (this *DeviceCenter) HostConnection(deviceId, hostId string) map[int32]*types.Connection {
	if this == nil {
		panic("nil")
	}
	filter := &types.Device{DeviceId: deviceId}
	d, _ := this.devices.Get(filter)
	device, _ := d.(*types.Device)
	if device == nil {
		return nil
	}
	return device.Hosts[hostId].Configs
}

func Configs(resource ifs.IResources, serviceArea byte) *DeviceCenter {
	sp, ok := resource.Services().ServiceHandler(ServiceName, serviceArea)
	if !ok {
		return nil
	}
	return (sp.(*DeviceService)).configCenter
}
