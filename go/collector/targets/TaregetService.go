package targets

/*
import (
	"github.com/saichler/l8collector/go/collector/common"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/web"
)

const (
	ServiceName = "Target"
	ServiceType = "TargetService"
)

type TargetService struct {
	configCenter *TargetCenter
	serviceArea  byte
}

func (this *TargetService) Activate(sla *ifs.ServiceLevelAgreement, vnic ifs.IVNic) error {
	vnic.Resources().Registry().Register(&l8tpollaris.L8PTarget{})
	vnic.Resources().Registry().Register(&l8tpollaris.L8PTargetList{})
	this.configCenter = newDeviceCenter(sla, vnic)
	this.serviceArea = sla.ServiceArea()
	return nil
}

func (this *TargetService) DeActivate() error {
	this.configCenter.Shutdown()
	this.configCenter = nil
	return nil
}

func (this *TargetService) Post(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	deviceList, ok := pb.Element().(*l8tpollaris.L8PTargetList)
	if ok {
		for _, device := range deviceList.List {
			ok = this.configCenter.Post(device, pb.Notification())
			if !ok {
				this.startDevice(device, vnic, pb.Notification())
			} else {
				this.updateDevice(device, vnic, pb.Notification())
			}
		}
	}
	device, ok := pb.Element().(*l8tpollaris.L8PTarget)
	if ok {
		ok = this.configCenter.Post(device, pb.Notification())
		if !ok {
			this.startDevice(device, vnic, pb.Notification())
		} else {
			this.updateDevice(device, vnic, pb.Notification())
		}
	}
	return object.New(nil, &l8tpollaris.L8PTarget{})
}

func (this *TargetService) Put(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	deviceList, ok := pb.Element().(*l8tpollaris.L8PTargetList)
	if ok {
		for _, device := range deviceList.List {
			ok = this.configCenter.Put(device, pb.Notification())
			if !ok {
				this.startDevice(device, vnic, pb.Notification())
			} else {
				this.updateDevice(device, vnic, pb.Notification())
			}
		}
	}
	device, ok := pb.Element().(*l8tpollaris.L8PTarget)
	if ok {
		ok = this.configCenter.Put(device, pb.Notification())
		if !ok {
			this.startDevice(device, vnic, pb.Notification())
		} else {
			this.updateDevice(device, vnic, pb.Notification())
		}
	}
	return object.New(nil, &l8tpollaris.L8PTarget{})
}
func (this *TargetService) Patch(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	deviceList, ok := pb.Element().(*l8tpollaris.L8PTargetList)
	if ok {
		for _, device := range deviceList.List {
			ok = this.configCenter.Patch(device, pb.Notification())
			if !ok {
				this.startDevice(device, vnic, pb.Notification())
			} else {
				this.updateDevice(device, vnic, pb.Notification())
			}
		}
	}
	device, ok := pb.Element().(*l8tpollaris.L8PTarget)
	if ok {
		ok = this.configCenter.Patch(device, pb.Notification())
		if !ok {
			this.startDevice(device, vnic, pb.Notification())
		} else {
			this.updateDevice(device, vnic, pb.Notification())
		}
	}
	return object.New(nil, &l8tpollaris.L8PTarget{})
}
func (this *TargetService) Delete(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	deviceList, ok := pb.Element().(*l8tpollaris.L8PTargetList)
	if ok {
		for _, device := range deviceList.List {
			ok = this.configCenter.Delete(device, pb.Notification())
			if ok {
				this.stopDevice(device, vnic, pb.Notification())
			}
		}
	}
	device, ok := pb.Element().(*l8tpollaris.L8PTarget)
	if ok {
		ok = this.configCenter.Delete(device, pb.Notification())
		if ok {
			this.stopDevice(device, vnic, pb.Notification())
		}
	}
	return object.New(nil, &l8tpollaris.L8PTarget{})
}

func (this *TargetService) Get(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *TargetService) GetCopy(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *TargetService) Failed(pb ifs.IElements, vnic ifs.IVNic, msg *ifs.Message) ifs.IElements {
	return nil
}
func (this *TargetService) TransactionConfig() ifs.ITransactionConfig {
	return nil
}

func (this *TargetService) WebService() ifs.IWebService {
	ws := web.New(ServiceName, this.serviceArea, &l8tpollaris.L8PTargetList{},
		&l8tpollaris.L8PTarget{}, nil, nil, nil, nil, nil, nil, nil, nil)
	return ws
}

func (this *TargetService) startDevice(device *l8tpollaris.L8PTarget, vnic ifs.IVNic, isNotificaton bool) {
	vnic.Resources().Logger().Info("TargetService.startDevice: ", device.TargetId)
	if !isNotificaton {
		err := vnic.RoundRobin(common.CollectorService, this.serviceArea, ifs.POST, device)
		if err != nil {
			vnic.Resources().Logger().Error("Device Service:", err.Error())
		}
	}
}

func (this *TargetService) updateDevice(device *l8tpollaris.L8PTarget, vnic ifs.IVNic, isNotificaton bool) {
	vnic.Resources().Logger().Info("TargetService.startDevice: ", device.TargetId)
	if !isNotificaton {
		err := vnic.Multicast(common.CollectorService, this.serviceArea, ifs.PUT, device)
		if err != nil {
			vnic.Resources().Logger().Error("Device Service:", " ", err.Error())
		}
	}
}

func (this *TargetService) stopDevice(device *l8tpollaris.L8PTarget, vnic ifs.IVNic, isNotificaton bool) {
	if !isNotificaton {
		err := vnic.Multicast(common.CollectorService, this.serviceArea, ifs.DELETE, device)
		if err != nil {
			vnic.Resources().Logger().Error("Device Service:", " ", err.Error())
		}
	}
}
*/
