/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
You may obtain a copy of the License at:

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package service

import (
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/web"
)

// ExecuteService handles remote job execution requests from other services.
// It allows jobs to be executed on-demand, either locally if the host collector
// exists, or by forwarding to another collector instance that owns the device.
//
// This service enables distributed collection where jobs can be triggered
// remotely for immediate execution, bypassing the normal scheduling cadence.
type ExecuteService struct {
	collectorService *CollectorService // Parent collector service
	serviceArea      byte              // Service area for routing
}

func (this *ExecuteService) Activate(sla *ifs.ServiceLevelAgreement, vnic ifs.IVNic) error {
	vnic.Resources().Registry().Register(&l8tpollaris.CJob{})
	this.collectorService = sla.Args()[0].(*CollectorService)
	this.serviceArea = sla.ServiceArea()
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
	job.Error = "Primary Not Found"
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
	ws := web.New("exec", this.serviceArea, 0)
	ws.AddEndpoint(&l8tpollaris.CJob{}, ifs.POST, &l8tpollaris.CJob{})
	return ws
}

func Exec(area byte, r ifs.IResources) *ExecuteService {
	s, ok := r.Services().ServiceHandler("exec", area)
	if ok {
		return s.(*ExecuteService)
	}
	return nil
}
