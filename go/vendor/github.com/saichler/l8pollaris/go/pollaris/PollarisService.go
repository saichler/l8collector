// © 2025 Sharon Aicler (saichler@gmail.com)
//
// Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pollaris

import (
	"errors"
	"github.com/saichler/l8parser/go/parser/boot"

	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8web"
	"github.com/saichler/l8utils/go/utils/web"
)

const (
	// ServiceName is the registered name of the Pollaris service in the service registry.
	ServiceName = "Pollaris"
	// ServiceArea defines the service area (partition) for the Pollaris service.
	// Area 0 indicates the default/global service area.
	ServiceArea = 0
)

// PollarisService implements the IServiceHandler interface for managing
// polling configurations. It provides CRUD operations for L8Pollaris objects
// and exposes web service endpoints for external access.
type PollarisService struct {
	// pollarisCenter is the central hub managing all polling configurations
	pollarisCenter *PollarisCenter
	// serviceArea stores the service area for this instance
	serviceArea byte
}

// Activate initializes and registers the Pollaris service with the VNic.
// It loads all predefined polling models from the boot package, creates a
// service level agreement, and activates the service in the service registry.
// This is the main entry point for starting the Pollaris service.
func Activate(vnic ifs.IVNic) error {
	initData := []interface{}{}
	for _, p := range boot.GetAllPolarisModels() {
		initData = append(initData, p)
	}
	initData = append(initData, boot.CreateK8sBootPolls())
	sla := ifs.NewServiceLevelAgreement(&PollarisService{}, ServiceName, ServiceArea, true, nil)
	sla.SetServiceItem(&l8tpollaris.L8Pollaris{})
	sla.SetInitItems(initData)
	vnic.Resources().Services().Activate(sla, vnic)
	return nil
}

// Activate is called by the service framework to initialize this service instance.
// It registers the L8Pollaris type with the registry and creates the PollarisCenter.
func (this *PollarisService) Activate(sla *ifs.ServiceLevelAgreement, vnic ifs.IVNic) error {
	vnic.Resources().Registry().Register(&l8tpollaris.L8Pollaris{})
	this.pollarisCenter = newPollarisCenter(sla, vnic)
	this.serviceArea = sla.ServiceArea()
	return nil
}

// DeActivate is called when the service is being shut down.
// It cleans up resources by setting the pollarisCenter to nil.
func (this *PollarisService) DeActivate() error {
	this.pollarisCenter = nil
	return nil
}

// Post handles creation of new L8Pollaris configurations.
// It iterates through the elements, validates each as L8Pollaris,
// and adds them to the PollarisCenter. Returns an empty response with
// any error that occurred during processing.
func (this *PollarisService) Post(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	var err error
	for _, elem := range pb.Elements() {
		l8Pollaris, ok := elem.(*l8tpollaris.L8Pollaris)
		if ok {
			vnic.Resources().Logger().Info("Added a l8Pollaris ", l8Pollaris.Name)
			e := this.pollarisCenter.Post(l8Pollaris, pb.Notification())
			if e != nil {
				err = e
			}
		} else {
			err = errors.New("Element is not a L8Pollaris")
		}
	}
	return object.New(err, &l8web.L8Empty{})
}
// Put handles updates to existing L8Pollaris configurations.
// Similar to Post, but uses the Put method on PollarisCenter which
// is semantically an update operation.
func (this *PollarisService) Put(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	var err error
	for _, elem := range pb.Elements() {
		l8Pollaris, ok := elem.(*l8tpollaris.L8Pollaris)
		if ok {
			vnic.Resources().Logger().Info("Added a l8Pollaris ", l8Pollaris.Name)
			e := this.pollarisCenter.Put(l8Pollaris, pb.Notification())
			if e != nil {
				err = e
			}
		} else {
			err = errors.New("Element is not a L8Pollaris")
		}
	}
	return object.New(err, &l8web.L8Empty{})
}
// Patch handles partial updates to L8Pollaris configurations.
// Currently not implemented - returns nil.
func (this *PollarisService) Patch(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}

// Delete handles deletion of L8Pollaris configurations.
// Currently not implemented - returns nil.
func (this *PollarisService) Delete(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}

// Get handles retrieval of L8Pollaris configurations.
// Currently not implemented - returns nil. Use PollarisByName or PollarisByKey instead.
func (this *PollarisService) Get(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}

// GetCopy returns a copy of the requested L8Pollaris configurations.
// Currently not implemented - returns nil.
func (this *PollarisService) GetCopy(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}

// Failed handles failed message delivery for L8Pollaris operations.
// Currently not implemented - returns nil.
func (this *PollarisService) Failed(pb ifs.IElements, vnic ifs.IVNic, msg *ifs.Message) ifs.IElements {
	return nil
}

// TransactionConfig returns the transaction configuration for this service.
// Currently not implemented - returns nil (no transaction support).
func (this *PollarisService) TransactionConfig() ifs.ITransactionConfig {
	return nil
}
// WebService returns the web service configuration for the Pollaris service.
// It exposes POST and PUT endpoints for L8Pollaris objects, allowing
// external clients to create and update polling configurations via HTTP.
func (this *PollarisService) WebService() ifs.IWebService {
	ws := web.New(ServiceName, ServiceArea, 0)
	ws.AddEndpoint(&l8tpollaris.L8Pollaris{}, ifs.POST, &l8web.L8Empty{})
	ws.AddEndpoint(&l8tpollaris.L8Pollaris{}, ifs.PUT, &l8web.L8Empty{})
	return ws
}
