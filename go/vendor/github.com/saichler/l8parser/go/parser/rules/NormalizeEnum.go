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
	"reflect"
	"strconv"
	"strings"

	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8reflect/go/reflect/properties"
	"github.com/saichler/l8types/go/ifs"
)

// NormalizeEnum is a parsing rule that normalizes raw SNMP values to protobuf enum values.
// It is designed to be chained after a Set rule. It reads the workspace Output (set by Set),
// applies a configurable value mapping, and re-sets the target property with the normalized value.
//
// Parameters:
//   - "map": comma-separated mapping of input:output pairs (e.g., "1:1,2:2,3:1,*:0")
//     The special key "*" serves as a default/fallback mapping for any unmapped value.
//     If no "*" mapping exists, unmapped values default to 0 (UNSPECIFIED).
type NormalizeEnum struct{}

// Name returns the rule identifier "NormalizeEnum".
func (this *NormalizeEnum) Name() string {
	return "NormalizeEnum"
}

// ParamNames returns the required parameter names for this rule.
func (this *NormalizeEnum) ParamNames() []string {
	return []string{"map"}
}

// Parse executes the NormalizeEnum rule logic.
func (this *NormalizeEnum) Parse(resources ifs.IResources, workSpace map[string]interface{}, params map[string]*l8tpollaris.L8PParameter, any interface{}, pollWhat string) error {
	output := workSpace[Output]
	if output == nil {
		return nil
	}

	_propertyId := workSpace[PropertyId]
	if _propertyId == nil {
		return nil
	}
	propertyId := _propertyId.(string)

	// Parse the value mapping from params
	mapParam, ok := params["map"]
	if !ok || mapParam == nil || mapParam.Value == "" {
		return resources.Logger().Error("NormalizeEnum: missing 'map' parameter")
	}
	valueMap, defaultVal := parseValueMap(mapParam.Value)

	// Convert the output value to an int64 for lookup
	inputKey, valid := toInt64(output)
	if !valid {
		// Non-numeric value (e.g., string) — use default
		inputKey = -1
	}

	// Look up the mapped enum value
	var enumVal int32
	if mapped, found := valueMap[inputKey]; found {
		enumVal = mapped
	} else {
		enumVal = defaultVal
	}

	// Set the normalized value on the target property
	modifiedPropertyId := injectIndexOrKey(propertyId, workSpace)
	instance, err := properties.PropertyOf(modifiedPropertyId, resources)
	if err != nil {
		return resources.Logger().Error("NormalizeEnum: error resolving property:", err.Error())
	}
	if instance != nil {
		_, _, err = instance.Set(any, enumVal)
		if err != nil {
			return resources.Logger().Error("NormalizeEnum: error setting property:", err.Error())
		}
	}

	workSpace[Output] = enumVal
	return nil
}

// parseValueMap parses a mapping string like "1:1,2:2,3:0,*:0" into a lookup map
// and a default value. Returns (map[inputVal]outputVal, defaultVal).
func parseValueMap(mapStr string) (map[int64]int32, int32) {
	result := make(map[int64]int32)
	var defaultVal int32

	pairs := strings.Split(mapStr, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		outVal, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			continue
		}

		if key == "*" {
			defaultVal = int32(outVal)
		} else {
			inVal, err := strconv.ParseInt(key, 10, 64)
			if err != nil {
				continue
			}
			result[inVal] = int32(outVal)
		}
	}

	return result, defaultVal
}

// toInt64 attempts to convert an interface{} value to int64.
func toInt64(value interface{}) (int64, bool) {
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int(), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(v.Uint()), true
	case reflect.Float32, reflect.Float64:
		return int64(v.Float()), true
	default:
		return 0, false
	}
}
