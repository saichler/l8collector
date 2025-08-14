package devices

import (
	"github.com/saichler/l8pollaris/go/types"
	"github.com/saichler/l8services/go/services/dcache"
	"github.com/saichler/l8types/go/ifs"
)

type DeviceCenter struct {
	devices ifs.IDistributedCache
}

func newDeviceCenter(serviceName string, serviceArea byte, resources ifs.IResources, listener ifs.IServiceCacheListener) *DeviceCenter {
	this := &DeviceCenter{}
	this.devices = dcache.NewDistributedCache(serviceName, serviceArea, "Device", resources.SysConfig().LocalUuid, listener, resources)
	return this
}

func (this *DeviceCenter) Shutdown() {
	this.devices = nil
}

func (this *DeviceCenter) Add(device *types.Device, isNotification bool) bool {
	exist := this.devices.Get(device.DeviceId)
	this.devices.Put(device.DeviceId, device, isNotification)
	if exist != nil {
		return true
	}
	return false
}

func (this *DeviceCenter) DeviceById(id string) *types.Device {
	device, _ := this.devices.Get(id).(*types.Device)
	return device
}

func (this *DeviceCenter) HostConnection(deviceId, hostId string) map[int32]*types.Connection {
	if this == nil {
		panic("nil")
	}
	device, _ := this.devices.Get(deviceId).(*types.Device)
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
