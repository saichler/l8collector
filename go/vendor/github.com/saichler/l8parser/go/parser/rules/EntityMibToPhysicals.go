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
	"reflect"
	"strconv"
	"strings"

	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	types2 "github.com/saichler/probler/go/types"
)

// EntityMibToPhysicals is a parsing rule that transforms SNMP Entity MIB data (entPhysicalTable)
// into NetworkDevice physical component structures. It parses chassis, modules, ports, fans,
// power supplies, and other physical entities from standard SNMP Entity MIB data.
type EntityMibToPhysicals struct{}

// Name returns the rule identifier "EntityMibToPhysicals".
func (this *EntityMibToPhysicals) Name() string {
	return "EntityMibToPhysicals"
}

// ParamNames returns the required parameter names for this rule.
func (this *EntityMibToPhysicals) ParamNames() []string {
	return []string{""}
}

// Entity MIB OID structure:
// .1.3.6.1.2.1.47.1.1.1.1.1 - entPhysicalIndex (not in table - used as row key)
// .1.3.6.1.2.1.47.1.1.1.1.2 - entPhysicalDescr
// .1.3.6.1.2.1.47.1.1.1.1.3 - entPhysicalVendorType
// .1.3.6.1.2.1.47.1.1.1.1.4 - entPhysicalContainedIn
// .1.3.6.1.2.1.47.1.1.1.1.5 - entPhysicalClass
// .1.3.6.1.2.1.47.1.1.1.1.6 - entPhysicalParentRelPos
// .1.3.6.1.2.1.47.1.1.1.1.7 - entPhysicalName
// .1.3.6.1.2.1.47.1.1.1.1.8 - entPhysicalHardwareRev
// .1.3.6.1.2.1.47.1.1.1.1.9 - entPhysicalFirmwareRev
// .1.3.6.1.2.1.47.1.1.1.1.10 - entPhysicalSoftwareRev
// .1.3.6.1.2.1.47.1.1.1.1.11 - entPhysicalSerialNum
// .1.3.6.1.2.1.47.1.1.1.1.12 - entPhysicalMfgName
// .1.3.6.1.2.1.47.1.1.1.1.13 - entPhysicalModelName
// .1.3.6.1.2.1.47.1.1.1.1.14 - entPhysicalAlias
// .1.3.6.1.2.1.47.1.1.1.1.15 - entPhysicalAssetID
// .1.3.6.1.2.1.47.1.1.1.1.16 - entPhysicalIsFRU

// Entity Physical Class enum values:
const (
	EntPhysicalClassOther       = 1
	EntPhysicalClassUnknown     = 2
	EntPhysicalClassChassis     = 3
	EntPhysicalClassBackplane   = 4
	EntPhysicalClassContainer   = 5
	EntPhysicalClassPowerSupply = 6
	EntPhysicalClassFan         = 7
	EntPhysicalClassSensor      = 8
	EntPhysicalClassModule      = 9
	EntPhysicalClassPort        = 10
	EntPhysicalClassStack       = 11
	EntPhysicalClassCpu         = 12
)

func (this *EntityMibToPhysicals) Parse(resources ifs.IResources, workSpace map[string]interface{}, params map[string]*l8tpollaris.L8PParameter, any interface{}, pollWhat string) error {
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

	// Create the main physical component
	physicalKey := "physical-0"
	physical, exists := networkDevice.Physicals[physicalKey]
	if !exists {
		physical = &types2.Physical{
			Id: physicalKey,
		}
		networkDevice.Physicals[physicalKey] = physical
	}

	// Initialize component maps
	portMap := make(map[string]*types2.Port)

	// First, collect entity data from all columns for each entity
	entityData := make(map[string]map[int]interface{}) // entityIndex -> column -> value

	// Process each row in the Entity MIB table
	for rowKey, row := range table.Rows {
		entityIndex := fmt.Sprintf("%d", rowKey)
		entityData[entityIndex] = make(map[int]interface{})

		// Handle multi-column Entity MIB table
		if len(table.Columns) > 1 {
			// Multi-column case - collect data from all relevant columns
			for colKey, colName := range table.Columns {
				if colNum, err := strconv.Atoi(colName); err == nil {
					// Collect data from key columns
					if colNum == 2 || colNum == 5 || colNum == 7 || colNum == 11 || colNum == 13 { // descr, class, name, serial, model
						data, ok := row.Data[colKey]
						if ok {
							value := getEntityValue(data, resources)
							if value != nil {
								entityData[entityIndex][colNum] = value
							}
						}
					}
				}
			}
		} else {
			// Single column case - get the column number from column name
			var entityColumn int
			for colKey, colName := range table.Columns {
				if colKey == 0 {
					if colNum, err := strconv.Atoi(colName); err == nil {
						entityColumn = colNum
					}
					break
				}
			}

			// For now, we'll focus on the key columns that define entity structure
			// Skip processing if this isn't one of the key columns we need
			if entityColumn != 2 && entityColumn != 5 && entityColumn != 7 && entityColumn != 11 && entityColumn != 13 { // descr, class, name, serial, model
				continue
			}

			// Get the value for this column
			data, ok := row.Data[0] // Data is always in column 0 for single-column CTable
			if !ok {
				continue
			}
			value := getEntityValue(data, resources)
			if value != nil {
				entityData[entityIndex][entityColumn] = value
			}
		}
	}

	// Now process the collected entity data to create ports and interfaces
	for entityIndex, columns := range entityData {
		// Check if this entity is a port (entPhysicalClass = 10)
		if classValue, exists := columns[5]; exists { // column 5 = entPhysicalClass
			entityClassInt := 0
			if classStr := fmt.Sprintf("%v", classValue); classStr != "" {
				// Handle case where value comes as "INTEGER: 10" format
				if strings.HasPrefix(classStr, "INTEGER: ") {
					classStr = strings.TrimPrefix(classStr, "INTEGER: ")
				}
				if val, err := strconv.Atoi(classStr); err == nil {
					entityClassInt = val
				}
			}
			if entityClassInt == EntPhysicalClassPort {
				// Create port with collected data
				port := &types2.Port{
					Id: entityIndex,
				}

				// Create an interface for this port using Entity MIB data
				iface := &types2.Interface{
					Id: entityIndex,
				}

				// Populate interface with Entity MIB data
				if nameValue, exists := columns[7]; exists { // column 7 = entPhysicalName
					// Handle case where value comes as "STRING: value" format
					nameStr := fmt.Sprintf("%v", nameValue)
					if strings.HasPrefix(nameStr, "STRING: ") {
						nameStr = strings.TrimPrefix(nameStr, "STRING: ")
						nameStr = strings.Trim(nameStr, `"`) // Remove quotes if present
						iface.Name = strings.TrimSpace(nameStr)
					} else {
						// Fallback to original conversion logic
						if name, err := convertToString(nameValue, reflect.Slice); err == nil {
							iface.Name = strings.TrimSpace(name)
						}
					}
				}

				if descrValue, exists := columns[2]; exists { // column 2 = entPhysicalDescr
					// Handle case where value comes as "STRING: value" format
					descrStr := fmt.Sprintf("%v", descrValue)
					if strings.HasPrefix(descrStr, "STRING: ") {
						descrStr = strings.TrimPrefix(descrStr, "STRING: ")
						descrStr = strings.Trim(descrStr, `"`) // Remove quotes if present
						iface.Description = strings.TrimSpace(descrStr)
					} else {
						// Fallback to original conversion logic
						if descr, err := convertToString(descrValue, reflect.Slice); err == nil {
							iface.Description = strings.TrimSpace(descr)
						}
					}
				}

				// Add interface to port
				port.Interfaces = make([]*types2.Interface, 1)
				port.Interfaces[0] = iface

				portMap[entityIndex] = port
			}
		}
	}

	// Convert maps to slices and assign to physical component
	if len(portMap) > 0 {
		physical.Ports = make([]*types2.Port, 0, len(portMap))
		for _, port := range portMap {
			physical.Ports = append(physical.Ports, port)
		}
	}

	return nil
}

func getEntityValue(data []byte, resources ifs.IResources) interface{} {
	if len(data) == 0 {
		return nil
	}
	obj := object.NewDecode(data, 0, resources.Registry())
	val, _ := obj.Get()
	return val
}

func getEntityStringValue(data []byte, resources ifs.IResources) string {
	val := getEntityValue(data, resources)
	if val == nil {
		return ""
	}

	// Handle byte array to string conversion
	if byteArray, ok := val.([]uint8); ok {
		return strings.TrimSpace(string(byteArray))
	}

	return strings.TrimSpace(fmt.Sprintf("%v", val))
}
