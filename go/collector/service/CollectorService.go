package service

import (
	"github.com/saichler/l8pollaris/go/types"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/maps"
	"github.com/saichler/l8utils/go/utils/strings"
)

const (
	ServiceType = "CollectorService"
)

type CollectorService struct {
	serviceArea    byte
	hostCollectors *maps.SyncMap
	vnic           ifs.IVNic
}

func (this *CollectorService) Activate(serviceName string, serviceArea byte,
	r ifs.IResources, l ifs.IServiceCacheListener, args ...interface{}) error {
	this.serviceArea = serviceArea
	this.hostCollectors = maps.NewSyncMap()
	vnic, ok := l.(ifs.IVNic)
	if ok {
		this.vnic = vnic
		r.Registry().Register(&types.Device{})
		r.Registry().Register(&types.CMap{})
		r.Registry().Register(&types.CTable{})
		r.Registry().Register(&types.CJob{})
		r.Registry().Register(&ExecuteService{})
		r.Services().Activate("ExecuteService", "exec", serviceArea, r, nil, this)
	}
	return nil
}

func (this *CollectorService) startPolling(device *types.Device) error {
	for _, host := range device.Hosts {
		hostCol, _ := this.hostCollector(host.DeviceId, device)
		hostCol.start()
	}
	return nil
}

func (this *CollectorService) hostCollector(hostId string, device *types.Device) (*HostCollector, bool) {
	key := hostCollectorKey(device.DeviceId, hostId)
	h, ok := this.hostCollectors.Get(key)
	if ok {
		return h.(*HostCollector), ok
	}
	hc := newHostCollector(device, hostId, this)
	this.hostCollectors.Put(key, hc)
	return hc, ok
}

func hostCollectorKey(deviceId, hostId string) string {
	return strings.New(deviceId, hostId).String()
}

func (this *CollectorService) DeActivate() error {
	this.vnic = nil
	return nil
}

func (this *CollectorService) Post(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	device := pb.Element().(*types.Device)
	vnic.Resources().Logger().Info("Collector Service: Start polling device ", device.DeviceId)
	this.startPolling(device)
	return object.New(nil, &types.Device{})
}
func (this *CollectorService) Put(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *CollectorService) Patch(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *CollectorService) Delete(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *CollectorService) Get(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *CollectorService) GetCopy(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *CollectorService) Failed(pb ifs.IElements, vnic ifs.IVNic, msg *ifs.Message) ifs.IElements {
	return nil
}
func (this *CollectorService) TransactionMethod() ifs.ITransactionMethod {
	return nil
}
func (this *CollectorService) WebService() ifs.IWebService {
	return nil
}
