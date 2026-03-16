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
	"strconv"

	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	types2 "github.com/saichler/probler/go/types"
)

// IfTableToPhysicals is a parsing rule that transforms SNMP IF-MIB interface table data
// into NetworkDevice physical port/interface structures. It creates interface objects
// with properties like name, description, type, speed, and operational status.
type IfTableToPhysicals struct{}

// Name returns the rule identifier "IfTableToPhysicals".
func (this *IfTableToPhysicals) Name() string {
	return "IfTableToPhysicals"
}

// ParamNames returns the required parameter names for this rule.
func (this *IfTableToPhysicals) ParamNames() []string {
	return []string{""}
}

// Parse executes the IfTableToPhysicals rule, converting IF-MIB data to physical structures.
func (this *IfTableToPhysicals) Parse(resources ifs.IResources, workSpace map[string]interface{}, params map[string]*l8tpollaris.L8PParameter, any interface{}, pollWhat string) error {
	// Get the input CTable from workspace
	input := workSpace[Input]
	if input == nil {
		return errors.New("No input data found in workspace")
	}

	// Try to get CTable from input
	table, ok := input.(*l8tpollaris.CTable)
	if !ok {
		return errors.New("Input is not a CTable: " + fmt.Sprintf("%T", input))
	}

	// Get the NetworkDevice
	networkDevice, ok := any.(*types2.NetworkDevice)
	if !ok {
		return errors.New("Target object is not a NetworkDevice")
	}

	// Ensure physicals map exists
	if networkDevice.Physicals == nil {
		networkDevice.Physicals = make(map[string]*types2.Physical)
	}

	// Create or get the main physical component
	physicalKey := "physical-0"
	physical, exists := networkDevice.Physicals[physicalKey]
	if !exists {
		physical = &types2.Physical{
			Id: physicalKey,
		}
		networkDevice.Physicals[physicalKey] = physical
	}

	// Ensure ports slice exists
	if physical.Ports == nil {
		physical.Ports = make([]*types2.Port, 0)
	}

	// Create a map to track existing ports by ID for efficient lookup
	portMap := make(map[string]*types2.Port)
	for _, port := range physical.Ports {
		portMap[port.Id] = port
	}

	// Process each row in the ifTable
	for rowKey, row := range table.Rows {
		// The ifIndex is likely the rowKey itself in CTable structure
		ifIndexStr := fmt.Sprintf("%d", rowKey)

		// Create port for this interface
		port, portExists := portMap[ifIndexStr]
		if !portExists {
			port = &types2.Port{
				Id: ifIndexStr,
			}
			portMap[ifIndexStr] = port
			physical.Ports = append(physical.Ports, port)
		}

		// Ensure interfaces slice exists
		if port.Interfaces == nil {
			port.Interfaces = make([]*types2.Interface, 0)
		}

		// Create interface
		iface := &types2.Interface{
			Id: ifIndexStr,
		}

		// Populate interface fields from ifTable columns (corrected mapping)
		// Column 2: ifDescr (interface name) - convert byte array to string
		if ifDescrData, ok := row.Data[2]; ok {
			if ifDescr := getIfTableValue(ifDescrData, resources); ifDescr != nil {
				if byteArray, ok := ifDescr.([]uint8); ok {
					iface.Name = string(byteArray)
				} else {
					iface.Name = fmt.Sprintf("%v", ifDescr)
				}
			}
		}

		// Column 8: ifOperStatus (interface status)
		if ifOperStatusData, ok := row.Data[8]; ok {
			if ifOperStatus := getIfTableValue(ifOperStatusData, resources); ifOperStatus != nil {
				iface.Status = fmt.Sprintf("%v", ifOperStatus)
			}
		}

		// Column 3: ifType (interface type)
		if ifTypeData, ok := row.Data[3]; ok {
			if ifType := getIfTableValue(ifTypeData, resources); ifType != nil {
				if typeInt, err := strconv.Atoi(fmt.Sprintf("%v", ifType)); err == nil {
					iface.InterfaceType = types2.InterfaceType(typeInt)
				}
			}
		}

		// Column 5: ifSpeed (interface speed)
		if ifSpeedData, ok := row.Data[5]; ok {
			if ifSpeed := getIfTableValue(ifSpeedData, resources); ifSpeed != nil {
				if speedInt, err := strconv.ParseUint(fmt.Sprintf("%v", ifSpeed), 10, 64); err == nil {
					iface.Speed = speedInt
				}
			}
		}

		// Column 4: ifMtu (interface MTU)
		if ifMtuData, ok := row.Data[4]; ok {
			if ifMtu := getIfTableValue(ifMtuData, resources); ifMtu != nil {
				if mtuInt, err := strconv.ParseUint(fmt.Sprintf("%v", ifMtu), 10, 32); err == nil {
					iface.Mtu = uint32(mtuInt)
				}
			}
		}

		// Column 6: ifPhysAddress (MAC address) - convert byte array to string
		if ifPhysAddrData, ok := row.Data[6]; ok {
			if ifPhysAddr := getIfTableValue(ifPhysAddrData, resources); ifPhysAddr != nil {
				if byteArray, ok := ifPhysAddr.([]uint8); ok {
					iface.MacAddress = string(byteArray)
				} else {
					iface.MacAddress = fmt.Sprintf("%v", ifPhysAddr)
				}
			}
		}

		// Column 7: ifAdminStatus (admin status)
		if ifAdminStatusData, ok := row.Data[7]; ok {
			if ifAdminStatus := getIfTableValue(ifAdminStatusData, resources); ifAdminStatus != nil {
				if adminInt, err := strconv.Atoi(fmt.Sprintf("%v", ifAdminStatus)); err == nil {
					iface.AdminStatus = adminInt == 1 // 1 = up, 2 = down
				}
			}
		}

		// Initialize statistics if interface statistics columns are present
		if hasStatistics(row.Data) {
			iface.Statistics = &types2.InterfaceStatistics{}

			// Column 10: ifInOctets
			if data, ok := row.Data[10]; ok {
				if val := getIfTableValue(data, resources); val != nil {
					if intVal, err := strconv.ParseUint(fmt.Sprintf("%v", val), 10, 64); err == nil {
						iface.Statistics.RxBytes = intVal
					}
				}
			}

			// Column 16: ifOutOctets
			if data, ok := row.Data[16]; ok {
				if val := getIfTableValue(data, resources); val != nil {
					if intVal, err := strconv.ParseUint(fmt.Sprintf("%v", val), 10, 64); err == nil {
						iface.Statistics.TxBytes = intVal
					}
				}
			}

			// Column 11: ifInUcastPkts
			if data, ok := row.Data[11]; ok {
				if val := getIfTableValue(data, resources); val != nil {
					if intVal, err := strconv.ParseUint(fmt.Sprintf("%v", val), 10, 64); err == nil {
						iface.Statistics.RxPackets = intVal
					}
				}
			}

			// Column 17: ifOutUcastPkts
			if data, ok := row.Data[17]; ok {
				if val := getIfTableValue(data, resources); val != nil {
					if intVal, err := strconv.ParseUint(fmt.Sprintf("%v", val), 10, 64); err == nil {
						iface.Statistics.TxPackets = intVal
					}
				}
			}

			// Column 14: ifInErrors
			if data, ok := row.Data[14]; ok {
				if val := getIfTableValue(data, resources); val != nil {
					if intVal, err := strconv.ParseUint(fmt.Sprintf("%v", val), 10, 64); err == nil {
						iface.Statistics.RxErrors = intVal
					}
				}
			}

			// Column 20: ifOutErrors
			if data, ok := row.Data[20]; ok {
				if val := getIfTableValue(data, resources); val != nil {
					if intVal, err := strconv.ParseUint(fmt.Sprintf("%v", val), 10, 64); err == nil {
						iface.Statistics.TxErrors = intVal
					}
				}
			}

			// Column 13: ifInDiscards
			if data, ok := row.Data[13]; ok {
				if val := getIfTableValue(data, resources); val != nil {
					if intVal, err := strconv.ParseUint(fmt.Sprintf("%v", val), 10, 64); err == nil {
						iface.Statistics.RxDrops = intVal
					}
				}
			}

			// Column 19: ifOutDiscards
			if data, ok := row.Data[19]; ok {
				if val := getIfTableValue(data, resources); val != nil {
					if intVal, err := strconv.ParseUint(fmt.Sprintf("%v", val), 10, 64); err == nil {
						iface.Statistics.TxDrops = intVal
					}
				}
			}
		}

		// Add interface to port
		port.Interfaces = append(port.Interfaces, iface)
	}
	return nil
}

func getIfTableValue(data []byte, resources ifs.IResources) interface{} {
	if len(data) == 0 {
		return nil
	}
	obj := object.NewDecode(data, 0, resources.Registry())
	val, _ := obj.Get()
	return val
}

func hasStatistics(data map[int32][]byte) bool {
	// Check if any of the statistics columns are present
	statsCols := []int32{10, 11, 13, 14, 16, 17, 19, 20}
	for _, col := range statsCols {
		if _, ok := data[col]; ok {
			return true
		}
	}
	return false
}
