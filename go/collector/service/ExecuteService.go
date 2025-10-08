package service

import (
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/web"
)

type ExecuteService struct {
	collectorService *CollectorService
	serviceArea      byte
}

func (this *ExecuteService) Activate(serviceName string, serviceArea byte,
	r ifs.IResources, l ifs.IServiceCacheListener, args ...interface{}) error {
	r.Registry().Register(&l8tpollaris.CJob{})
	this.collectorService = args[0].(*CollectorService)
	this.serviceArea = serviceArea
	return nil
}

func (this *ExecuteService) DeActivate() error {
	this.collectorService = nil
	return nil
}

func (this *ExecuteService) Post(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	job := pb.Element().(*l8tpollaris.CJob)
	key := hostCollectorKey(job.TargetId, job.HostId)
	h, ok := this.collectorService.hostCollectors.Get(key)
	if ok {
		hostController := h.(*HostCollector)
		hostController.execJob(job)
		return object.New(nil, job)
	} else {
		uuids := vnic.Resources().Services().GetParticipants("exec", this.serviceArea)
		delete(uuids, vnic.Resources().SysConfig().LocalUuid)
		for uuid, _ := range uuids {
			resp := vnic.Request(uuid, "exec", this.serviceArea, ifs.PUT, job, 30)
			if resp.Error() == nil {
				return resp
			}
		}
	}

	return object.New(nil, job)
}

func (this *ExecuteService) Put(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	job := pb.Element().(*l8tpollaris.CJob)
	key := hostCollectorKey(job.TargetId, job.HostId)
	h, ok := this.collectorService.hostCollectors.Get(key)
	if ok {
		hostController := h.(*HostCollector)
		hostController.execJob(job)
		return object.New(nil, job)
	}
	return object.NewError("No job was found with key: " + key)
}
func (this *ExecuteService) Patch(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *ExecuteService) Delete(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *ExecuteService) Get(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *ExecuteService) GetCopy(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *ExecuteService) Failed(pb ifs.IElements, vnic ifs.IVNic, msg *ifs.Message) ifs.IElements {
	return nil
}
func (this *ExecuteService) TransactionConfig() ifs.ITransactionConfig {
	return nil
}
func (this *ExecuteService) WebService() ifs.IWebService {
	ws := web.New("exec", this.serviceArea, &l8tpollaris.CJob{},
		&l8tpollaris.CJob{}, nil, nil, nil, nil, nil, nil, nil, nil)
	return ws
}

func Exec(area byte, r ifs.IResources) *ExecuteService {
	s, ok := r.Services().ServiceHandler("exec", area)
	if ok {
		return s.(*ExecuteService)
	}
	return nil
}
