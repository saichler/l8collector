package devices

import (
	"github.com/saichler/l8collector/go/collector/common"
	"github.com/saichler/l8pollaris/go/types"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/web"
)

const (
	ServiceName = "Devices"
	ServiceType = "DeviceService"
)

type DeviceService struct {
	configCenter *DeviceCenter
	serviceArea  byte
}

func (this *DeviceService) Activate(serviceName string, serviceArea byte,
	r ifs.IResources, l ifs.IServiceCacheListener, args ...interface{}) error {
	r.Registry().Register(&types.Device{})
	this.configCenter = newDeviceCenter(ServiceName, serviceArea, r, l)
	this.serviceArea = serviceArea
	return nil
}

func (this *DeviceService) DeActivate() error {
	this.configCenter.Shutdown()
	this.configCenter = nil
	return nil
}

func (this *DeviceService) Post(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	device, ok := pb.Element().(*types.Device)
	if ok {
		this.addDevice(device, vnic, pb.Notification())
	}
	devices, ok := pb.Element().(*types.DeviceList)
	if ok {
		for _, device = range devices.List {
			this.addDevice(device, vnic, pb.Notification())
		}
	}
	return object.New(nil, &types.Device{})
}

func (this *DeviceService) addDevice(device *types.Device, vnic ifs.IVNic, isNotificaton bool) {
	vnic.Resources().Logger().Info("Device Service: Added Device ", device.DeviceId)
	exist := this.configCenter.Add(device, isNotificaton)
	if !isNotificaton {
		if !exist {
			err := vnic.RoundRobin(common.CollectorService, this.serviceArea, ifs.POST, device)
			if err != nil {
				vnic.Resources().Logger().Error("Device Service:", err.Error())
			}
		} else {
			err := vnic.Multicast(common.CollectorService, this.serviceArea, ifs.PUT, device)
			if err != nil {
				vnic.Resources().Logger().Error("Device Service:", " ", err.Error())
			}
		}
	}
}

func (this *DeviceService) Put(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *DeviceService) Patch(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *DeviceService) Delete(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *DeviceService) Get(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *DeviceService) GetCopy(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *DeviceService) Failed(pb ifs.IElements, vnic ifs.IVNic, msg *ifs.Message) ifs.IElements {
	return nil
}
func (this *DeviceService) TransactionConfig() ifs.ITransactionConfig {
	return nil
}
func (this *DeviceService) WebService() ifs.IWebService {
	ws := web.New(ServiceName, this.serviceArea, &types.Device{},
		&types.Device{}, nil, nil, nil, nil, nil, nil, nil, nil)
	return ws
}
