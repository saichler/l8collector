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

package rules

import (
	"errors"
	"fmt"
	"strings"

	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	types2 "github.com/saichler/probler/go/types"
)

// SnmpOspfToVrf is a bulk parsing rule that transforms OSPF MIB (1.3.6.1.2.1.14) walk results
// into a VrfInstance.OspfInfo structure on the NetworkDevice model.
// It extracts general OSPF parameters, area table, interface table, and neighbor table data.
type SnmpOspfToVrf struct{}

// Name returns the rule identifier "SnmpOspfToVrf".
func (this *SnmpOspfToVrf) Name() string {
	return "SnmpOspfToVrf"
}

// ParamNames returns the required parameter names for this rule.
func (this *SnmpOspfToVrf) ParamNames() []string {
	return []string{}
}

// Parse executes the SnmpOspfToVrf rule, building OspfInfo from OSPF MIB walk data.
func (this *SnmpOspfToVrf) Parse(resources ifs.IResources, workSpace map[string]interface{}, params map[string]*l8tpollaris.L8PParameter, any interface{}, pollWhat string) error {
	input := workSpace[Input]
	if input == nil {
		return errors.New("SnmpOspfToVrf: no input data")
	}

	cmap, ok := input.(*l8tpollaris.CMap)
	if !ok {
		return errors.New("SnmpOspfToVrf: input is not a CMap")
	}

	if len(cmap.Data) == 0 {
		return nil // No OSPF data available
	}

	networkDevice, ok := any.(*types2.NetworkDevice)
	if !ok {
		return errors.New("SnmpOspfToVrf: target is not a NetworkDevice")
	}

	ospfInfo := &types2.OspfInfo{}

	// Extract general OSPF params (1.3.6.1.2.1.14.1.*)
	routerId := ospfGetString(cmap, ".1.3.6.1.2.1.14.1.1.0", resources)
	if routerId == "" {
		return nil // OSPF not running on this device
	}

	ospfInfo.OspfEnabled = true
	ospfInfo.RouterId = routerId

	adminStat := ospfGetInt64(cmap, ".1.3.6.1.2.1.14.1.2.0", resources)
	if adminStat == 2 {
		ospfInfo.OspfEnabled = false
	}

	// Extract area info — find first area from area table (14.2.1.1.<area>)
	ospfInfo.AreaId = ospfFindFirstArea(cmap, resources)

	// Extract interface info — use first interface for cost and priority
	ospfExtractInterfaceInfo(cmap, ospfInfo, resources)

	// Extract neighbors from neighbor table (14.10.1.*)
	ospfInfo.Neighbors = ospfExtractNeighbors(cmap, resources)

	// Set on NetworkDevice
	ensureLogicalVrf(networkDevice)
	vrf := networkDevice.Logicals["logical-0"].Vrfs[0]
	vrf.OspfInfo = ospfInfo

	return nil
}

// ospfFindFirstArea scans the CMap for the first ospfAreaId entry (14.2.1.1.<area>).
func ospfFindFirstArea(cmap *l8tpollaris.CMap, resources ifs.IResources) string {
	prefix := ".1.3.6.1.2.1.14.2.1.1."
	for key := range cmap.Data {
		if strings.HasPrefix(key, prefix) {
			val := ospfGetString(cmap, key, resources)
			if val != "" {
				return val
			}
		}
	}
	return ""
}

// ospfExtractInterfaceInfo extracts cost and priority from the first OSPF interface entry.
func ospfExtractInterfaceInfo(cmap *l8tpollaris.CMap, info *types2.OspfInfo, resources ifs.IResources) {
	// Find first interface IP from interface table (14.7.1.1.<ip>)
	costPrefix := ".1.3.6.1.2.1.14.7.1.8."
	priorityPrefix := ".1.3.6.1.2.1.14.7.1.6."
	transitDelayPrefix := ".1.3.6.1.2.1.14.7.1.9."
	statePrefix := ".1.3.6.1.2.1.14.7.1.12."

	for key := range cmap.Data {
		if strings.HasPrefix(key, costPrefix) {
			ip := strings.TrimPrefix(key, costPrefix)
			info.Cost = uint32(ospfGetInt64(cmap, costPrefix+ip, resources))
			info.Priority = uint32(ospfGetInt64(cmap, priorityPrefix+ip, resources))
			info.RetransmitInterval = uint32(ospfGetInt64(cmap, transitDelayPrefix+ip, resources))

			// Map interface state to network type
			ifState := ospfGetInt64(cmap, statePrefix+ip, resources)
			if ifState == 4 { // point-to-point
				info.NetworkType = types2.OspfNetworkType(1) // POINT_TO_POINT
			}
			break
		}
	}
}

// ospfExtractNeighbors builds OspfNeighbor entries from the neighbor table (14.10.1.*).
func ospfExtractNeighbors(cmap *l8tpollaris.CMap, resources ifs.IResources) []*types2.OspfNeighbor {
	neighbors := make([]*types2.OspfNeighbor, 0)

	// Find all neighbor IPs from ospfNbrIpAddr (14.10.1.1.<ip>.<idx>)
	nbrIpPrefix := ".1.3.6.1.2.1.14.10.1.1."
	nbrRtrIdPrefix := ".1.3.6.1.2.1.14.10.1.3."
	nbrStatePrefix := ".1.3.6.1.2.1.14.10.1.6."

	for key := range cmap.Data {
		if !strings.HasPrefix(key, nbrIpPrefix) {
			continue
		}
		suffix := strings.TrimPrefix(key, nbrIpPrefix) // e.g., "10.1.1.2.0"

		nbr := &types2.OspfNeighbor{}
		nbr.NeighborIp = ospfGetString(cmap, nbrIpPrefix+suffix, resources)
		nbr.NeighborId = ospfGetString(cmap, nbrRtrIdPrefix+suffix, resources)

		// Map SNMP neighbor state (1-8) to protobuf OspfNeighborState (1-8, 0=unknown)
		snmpState := ospfGetInt64(cmap, nbrStatePrefix+suffix, resources)
		if snmpState >= 1 && snmpState <= 8 {
			nbr.State = types2.OspfNeighborState(snmpState)
		}

		neighbors = append(neighbors, nbr)
	}

	return neighbors
}

// Helper functions for extracting typed values from CMap

func ospfGetString(cmap *l8tpollaris.CMap, key string, resources ifs.IResources) string {
	data := cmap.Data[key]
	if data == nil || len(data) == 0 {
		return ""
	}
	enc := object.NewDecode(data, 0, resources.Registry())
	val, err := enc.Get()
	if err != nil || val == nil {
		return ""
	}
	if s, ok := val.(string); ok {
		if isSnmpErrorString(s) {
			return ""
		}
		return s
	}
	if b, ok := val.([]byte); ok {
		return string(b)
	}
	return fmt.Sprintf("%v", val)
}

func ospfGetInt64(cmap *l8tpollaris.CMap, key string, resources ifs.IResources) int64 {
	data := cmap.Data[key]
	if data == nil || len(data) == 0 {
		return 0
	}
	enc := object.NewDecode(data, 0, resources.Registry())
	val, err := enc.Get()
	if err != nil || val == nil {
		return 0
	}
	return toInt64Value(val)
}

// ensureLogicalVrf ensures the NetworkDevice has a logical-0 entry with at least one VrfInstance.
func ensureLogicalVrf(nd *types2.NetworkDevice) {
	if nd.Logicals == nil {
		nd.Logicals = make(map[string]*types2.Logical)
	}
	logical, exists := nd.Logicals["logical-0"]
	if !exists {
		logical = &types2.Logical{Id: "logical-0"}
		nd.Logicals["logical-0"] = logical
	}
	if logical.Vrfs == nil || len(logical.Vrfs) == 0 {
		logical.Vrfs = []*types2.VrfInstance{{VrfName: "default", Status: types2.VrfStatus(1)}}
	}
}

// toInt64Value converts any numeric interface{} to int64.
func toInt64Value(val interface{}) int64 {
	v, ok := toInt64(val)
	if ok {
		return v
	}
	return 0
}
