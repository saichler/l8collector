package service

import (
	"github.com/saichler/l8pollaris/go/pollaris/targets"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/maps"
	"github.com/saichler/l8utils/go/utils/strings"
)

type CollectorService struct {
	hostCollectors *maps.SyncMap
	vnic           ifs.IVNic
}

func Activate(linksID string, vnic ifs.IVNic) {
	collServiceName, collServiceArea := targets.Links.Collector(linksID)
	vnic.Resources().Logger().Info("Starting Collector on ", collServiceName, " area ", collServiceArea)
	sla := ifs.NewServiceLevelAgreement(&CollectorService{}, collServiceName, collServiceArea, true, nil)
	vnic.Resources().Services().Activate(sla, vnic)
}

func (this *CollectorService) Activate(sla *ifs.ServiceLevelAgreement, vnic ifs.IVNic) error {
	this.hostCollectors = maps.NewSyncMap()
	this.vnic = vnic
	vnic.Resources().Registry().Register(&l8tpollaris.L8PTarget{})
	vnic.Resources().Registry().Register(&l8tpollaris.CMap{})
	vnic.Resources().Registry().Register(&l8tpollaris.CTable{})
	vnic.Resources().Registry().Register(&l8tpollaris.CJob{})

	slaExec := ifs.NewServiceLevelAgreement(&ExecuteService{}, "exec", sla.ServiceArea(), false, nil)
	slaExec.SetArgs(this)
	vnic.Resources().Services().Activate(slaExec, vnic)

	return nil
}

func (this *CollectorService) startPolling(device *l8tpollaris.L8PTarget) error {
	for _, host := range device.Hosts {
		hostCol, _ := this.hostCollector(host.HostId, device)
		err := hostCol.start()
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *CollectorService) stopPolling(device *l8tpollaris.L8PTarget) {
	for _, host := range device.Hosts {
		key := hostCollectorKey(device.TargetId, host.HostId)
		h, ok := this.hostCollectors.Get(key)
		if ok {
			h.(*HostCollector).stop()
			this.hostCollectors.Delete(key)
		}
	}
}

func (this *CollectorService) hostCollector(hostId string, target *l8tpollaris.L8PTarget) (*HostCollector, bool) {
	key := hostCollectorKey(target.TargetId, hostId)
	h, ok := this.hostCollectors.Get(key)
	if ok {
		return h.(*HostCollector), ok
	}
	hc := newHostCollector(target, hostId, this)
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
	device := pb.Element().(*l8tpollaris.L8PTarget)
	switch device.State {
	case l8tpollaris.L8PTargetState_Up:
		vnic.Resources().Logger().Info("Collector Service: Start polling device ", device.TargetId)
		err := this.startPolling(device)
		if err != nil {
			vnic.Resources().Logger().Error("Collector Service: Error starting polling device ", device.TargetId)
			vnic.Resources().Logger().Error(err.Error())
		}
	case l8tpollaris.L8PTargetState_Down:
		vnic.Resources().Logger().Info("Collector Service: Stop polling device ", device.TargetId)
		this.stopPolling(device)
	}
	return object.New(nil, &l8tpollaris.L8PTarget{})
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
func (this *CollectorService) TransactionConfig() ifs.ITransactionConfig {
	return nil
}
func (this *CollectorService) WebService() ifs.IWebService {
	return nil
}
