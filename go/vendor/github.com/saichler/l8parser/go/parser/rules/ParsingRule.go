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

// Package rules provides the parsing rule engine for L8Parser.
// It defines the ParsingRule interface and implements various rule types
// for data transformation including Contains, Set, StringToCTable, CTableToMapProperty,
// EntityMibToPhysicals, IfTableToPhysicals, InferDeviceType, and MapToDeviceStatus.
package rules

import (
	"errors"
	"reflect"
	"strconv"
	"strings"

	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	l8strings "github.com/saichler/l8utils/go/utils/strings"
)

// ParsingRule defines the interface for all parsing rules in the L8Parser.
// Each rule must provide a name, parameter names, and a Parse method that
// transforms input data according to the rule's logic.
type ParsingRule interface {
	// Name returns the unique identifier for this rule type.
	Name() string
	// ParamNames returns the list of parameter names this rule expects.
	ParamNames() []string
	// Parse executes the rule logic, transforming input data and storing results in the workspace.
	// Parameters: resources (system resources), workspace (rule workspace), params (rule parameters),
	// input (data to parse), pollWhat (the poll identifier).
	Parse(ifs.IResources, map[string]interface{}, map[string]*l8tpollaris.L8PParameter, interface{}, string) error
}

func convertToString(value interface{}, kind reflect.Kind) (string, error) {
	switch kind {
	case reflect.String:
		return value.(string), nil
	case reflect.Slice:
		if byts, ok := value.([]byte); ok {
			return string(byts), nil
		}
		return "", errors.New("slice is not []byte")
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(reflect.ValueOf(value).Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(reflect.ValueOf(value).Uint(), 10), nil
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(reflect.ValueOf(value).Float(), 'f', -1, 64), nil
	case reflect.Bool:
		return strconv.FormatBool(value.(bool)), nil
	default:
		return "", errors.New("unsupported type for string conversion: " + kind.String())
	}
}

// GetValueInput extracts any value type from input data and returns the value, its reflect.Kind, and any error
func GetValueInput(resources ifs.IResources, input interface{}, params map[string]*l8tpollaris.L8PParameter, pollWhat string) (interface{}, reflect.Kind, error) {
	m, ok := input.(*l8tpollaris.CMap)
	if ok {
		if len(m.Data) == 0 {
			return nil, reflect.Invalid, errors.New("no data found in map:" + pollWhat)
		}
		from := params[From]
		if from == nil {
			return nil, reflect.Invalid, errors.New("missing 'from' key in map input")
		}
		rawData := m.Data[from.Value]
		if rawData == nil || len(rawData) == 0 {
			return nil, reflect.Invalid, errors.New("Value for From " + from.Value + " is blank")
		}
		enc := object.NewDecode(rawData, 0, resources.Registry())
		value, err := enc.Get()
		if err != nil {
			return nil, reflect.Invalid, errors.New("failed to decode value: " + err.Error())
		}
		if value == nil {
			return nil, reflect.Invalid, errors.New("failed to decode value")
		}
		return value, reflect.TypeOf(value).Kind(), nil
	}

	byts, ok := input.([]byte)
	if ok {
		return byts, reflect.Slice, nil
	}

	// Handle direct values
	if input != nil {
		return input, reflect.TypeOf(input).Kind(), nil
	}

	return nil, reflect.Invalid, errors.New("unsupported input type")
}

// injectIndexOrKey injects slice indices or map keys into PropertyId paths
// Format: <{reflect.Kind}value> before the attribute that needs indexing
func injectIndexOrKey(propertyId string, workSpace map[string]interface{}) string {
	// Map of collection attributes that need indexing/keying
	collectionMappings := map[string]string{
		"physicals":      "{24}physical-0", // map<string, Physical> - use string key
		"logicals":       "{24}logical-0",  // map<string, Logical> - use string key
		"networklinks":   "{2}0",           // repeated NetworkLink - use int index (alt name)
		"network_links":  "{2}0",           // repeated NetworkLink - use int index
		"chassis":        "{2}0",           // repeated Chassis - use int index
		"ports":          "{2}0",           // repeated Port - use int index
		"power_supplies": "{2}0",           // repeated PowerSupply - use int index
		"powersupplies":  "{2}0",           // repeated PowerSupply - use int index (alt name)
		"fans":           "{2}0",           // repeated Fan - use int index
		"modules":        "{2}0",           // repeated Module - use int index
		"cpus":           "{2}0",           // repeated Cpu - use int index
		"memorymodules":  "{2}0",           // repeated Memory - use int index
		"interfaces":     "{2}0",           // repeated Interface - use int index
		"processes":      "{2}0",           // repeated ProcessInfo - use int index
		"neighbors":      "{2}0",           // repeated OspfNeighbor - use int index
		"peers":          "{2}0",           // repeated BgpPeer - use int index
		"lsas":              "{2}0",           // repeated OspfLsa - use int index
		"routes":            "{2}0",           // repeated BgpRoute/VrfRoute - use int index
		"gpus":              "{24}gpu-0",      // map<string, Gpu> - use string key (GpuDevice)
		"networkinterfaces": "{2}0",           // repeated GpuNetworkInterface - use int index (GpuDeviceSystem)
		"gpu_links":         "{2}0",           // repeated GpuLink - use int index (GpuTopology)
		"checks":            "{2}0",           // repeated GpuHealthCheck - use int index (GpuDeviceHealth)
	}

	// Field name mappings for proto compatibility
	fieldMappings := map[string]string{
		"powersupplies": "powersupplies", // powersupplies -> power_supplies
		"networklinks":  "networklinks",  // networklinks -> network_links
		"networkhealth": "networkhealth", // networkhealth -> network_health
	}

	parts := strings.Split(propertyId, ".")
	result := make([]string, 0, len(parts))

	for i, part := range parts {
		// Apply field name mapping first
		mappedPart := part
		if mapped, exists := fieldMappings[part]; exists {
			mappedPart = mapped
		}

		// Apply collection indexing
		if indexKey, exists := collectionMappings[part]; exists {
			// Check if this is not the last part (we need a following attribute)
			if i < len(parts)-1 {
				// Inject the index/key before the next attribute
				result = append(result, mappedPart+"<"+indexKey+">")
			} else {
				result = append(result, mappedPart)
			}
		} else {
			result = append(result, mappedPart)
		}
	}

	modifiedId := strings.Join(result, ".")

	return modifiedId
}

// isSnmpErrorString checks if a string value is an SNMP error indicator
// rather than actual data. Devices that don't support a particular OID
// return these strings instead of values.
func isSnmpErrorString(s string) bool {
	lower := strings.ToLower(s)
	return strings.Contains(lower, "oid not supported") ||
		strings.Contains(lower, "no such object") ||
		strings.Contains(lower, "no such instance") ||
		strings.Contains(lower, "nosuchobject") ||
		strings.Contains(lower, "nosuchinstance")
}

func getIntInput(workSpace map[string]interface{}, paramName string) (int, error) {
	v, ok := workSpace[paramName].(string)
	if !ok {
		return -1, errors.New("'" + paramName + "' does not exist")
	}
	i, e := strconv.Atoi(v)
	if e != nil {
		return -1, e
	}
	return i, nil
}

func getIntArrInput(workSpace map[string]interface{}, paramName string) ([]int, error) {
	v, ok := workSpace[paramName].(string)
	if !ok {
		return []int{}, errors.New("'" + paramName + "' does not exist")
	}
	arr, e := l8strings.FromString(v, nil)
	if e != nil {
		return []int{}, e
	}
	return arr.Interface().([]int), nil
}
