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

// Package service provides the core collector service implementation that
// manages data collection from network devices. It handles service lifecycle,
// device target management, and coordinates host collectors for multi-protocol
// data collection.
package service

import (
	"github.com/saichler/l8pollaris/go/pollaris/targets"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/maps"
	"github.com/saichler/l8utils/go/utils/strings"
)

// CollectorService is the main service that manages data collection from
// network devices. It implements the Layer8 IService interface and coordinates
// multiple HostCollectors, one per target host.
//
// The service handles:
//   - Service activation and deactivation lifecycle
//   - Device target state changes (Up/Down)
//   - Host collector creation and management
//   - Protocol type registration
//
// CollectorService receives L8PTarget messages via the Post method to
// start or stop polling for specific devices.
type CollectorService struct {
	hostCollectors *maps.SyncMap  // Map of hostId -> HostCollector
	vnic           ifs.IVNic      // Virtual network interface for messaging
}

// Activate is the entry point for starting the CollectorService.
// It retrieves the collector configuration from the Links registry and
// activates the service with a Service Level Agreement (SLA).
//
// Parameters:
//   - linksID: The links identifier for looking up collector configuration
//   - vnic: The virtual network interface for service communication
func Activate(linksID string, vnic ifs.IVNic) {
	collServiceName, collServiceArea := targets.Links.Collector(linksID)
	vnic.Resources().Logger().Info("Starting Collector on ", collServiceName, " area ", collServiceArea)
	sla := ifs.NewServiceLevelAgreement(&CollectorService{}, collServiceName, collServiceArea, true, nil)
	vnic.Resources().Services().Activate(sla, vnic)
}

// Activate initializes the CollectorService when the service is activated.
// It sets up the host collectors map, registers required protobuf types,
// and activates the ExecuteService for handling remote job execution.
//
// Registered types:
//   - L8PTarget: Device target configuration
//   - CMap: Map-based collection results
//   - CTable: Table-based collection results
//   - CJob: Collection job structure
//
// Returns:
//   - Always returns nil (activation cannot fail)
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

// startPolling initiates data collection for all hosts in a device target.
// It creates or retrieves a HostCollector for each host and starts the
// collection process.
//
// Parameters:
//   - device: The L8PTarget containing host configurations
//
// Returns:
//   - error if any host collector fails to start
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

// stopPolling stops data collection for all hosts in a device target.
// It stops each HostCollector and removes it from the collectors map.
//
// Parameters:
//   - device: The L8PTarget containing host configurations to stop
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

// hostCollector retrieves or creates a HostCollector for the specified host.
// If a collector already exists, it returns the existing one. Otherwise,
// it creates a new HostCollector and stores it in the map.
//
// Parameters:
//   - hostId: The unique identifier for the host
//   - target: The L8PTarget containing the host configuration
//
// Returns:
//   - The HostCollector for the specified host
//   - bool indicating if the collector already existed
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

// hostCollectorKey generates a unique key for storing HostCollectors in the map.
// The key is a concatenation of the device ID and host ID.
func hostCollectorKey(deviceId, hostId string) string {
	return strings.New(deviceId, hostId).String()
}

// DeActivate is called when the service is being shut down.
// It releases the virtual network interface reference.
//
// Returns:
//   - Always returns nil
func (this *CollectorService) DeActivate() error {
	this.vnic = nil
	return nil
}

// Post handles incoming L8PTarget messages to start or stop device polling.
// When a device state changes to Up, polling is started. When it changes
// to Down, polling is stopped and resources are released.
//
// Parameters:
//   - pb: IElements containing the L8PTarget message
//   - vnic: The virtual network interface
//
// Returns:
//   - Empty L8PTarget response
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

// Put is not implemented for CollectorService.
func (this *CollectorService) Put(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}

// Patch is not implemented for CollectorService.
func (this *CollectorService) Patch(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}

// Delete is not implemented for CollectorService.
func (this *CollectorService) Delete(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}

// Get is not implemented for CollectorService.
func (this *CollectorService) Get(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}

// GetCopy is not implemented for CollectorService.
func (this *CollectorService) GetCopy(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}

// Failed handles failed message delivery for CollectorService.
func (this *CollectorService) Failed(pb ifs.IElements, vnic ifs.IVNic, msg *ifs.Message) ifs.IElements {
	return nil
}

// TransactionConfig returns nil as CollectorService doesn't use transactions.
func (this *CollectorService) TransactionConfig() ifs.ITransactionConfig {
	return nil
}

// WebService returns nil as CollectorService doesn't expose a web interface.
func (this *CollectorService) WebService() ifs.IWebService {
	return nil
}
