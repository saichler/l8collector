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
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8reflect/go/reflect/properties"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
)

// RestJsonParse is a parsing rule that extracts fields from JSON REST API responses
// using dot-path notation and maps them to target properties.
//
// Parameters:
//   - "mapping": comma-separated "jsonPath:propertyId" pairs
//     Example: "overall_health:gpudevice.health.overallstatus,nvlink_version:gpudevice.topology.nvlinkversion"
type RestJsonParse struct{}

// Name returns the rule identifier "RestJsonParse".
func (this *RestJsonParse) Name() string {
	return "RestJsonParse"
}

// ParamNames returns the required parameter names for this rule.
func (this *RestJsonParse) ParamNames() []string {
	return []string{"mapping"}
}

// Parse executes the RestJsonParse rule logic.
func (this *RestJsonParse) Parse(resources ifs.IResources, workSpace map[string]interface{},
	params map[string]*l8tpollaris.L8PParameter, any interface{}, pollWhat string) error {

	input := workSpace[Input]
	if input == nil {
		return nil
	}

	// Extract JSON string from CMap (key "json") sent by RestCollector
	var jsonStr string
	if cmap, ok := input.(*l8tpollaris.CMap); ok {
		jsonBytes, exists := cmap.Data["json"]
		if !exists || len(jsonBytes) == 0 {
			return errors.New("RestJsonParse: CMap has no 'json' key")
		}
		dec := object.NewDecode(jsonBytes, 0, resources.Registry())
		val, err := dec.Get()
		if err != nil {
			return errors.New("RestJsonParse: failed to decode json from CMap: " + err.Error())
		}
		jsonStr, _ = val.(string)
	} else if s, ok := input.(string); ok {
		jsonStr = s
	} else {
		return errors.New("RestJsonParse: unsupported input type: " + fmt.Sprintf("%T", input))
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &jsonData); err != nil {
		return errors.New("RestJsonParse: failed to parse JSON: " + err.Error())
	}

	if len(jsonData) == 0 {
		return nil
	}

	mappingParam := params["mapping"]
	if mappingParam == nil || mappingParam.Value == "" {
		return errors.New("RestJsonParse: missing 'mapping' parameter")
	}

	// Parse mapping entries: "jsonPath:propertyId,jsonPath:propertyId,..."
	entries := strings.Split(mappingParam.Value, ",")
	for _, entry := range entries {
		parts := strings.SplitN(strings.TrimSpace(entry), ":", 2)
		if len(parts) != 2 {
			continue
		}
		jsonPath := strings.TrimSpace(parts[0])
		targetPropId := strings.TrimSpace(parts[1])

		value := getJsonValue(jsonData, jsonPath)
		if value == nil {
			continue
		}

		// Handle array values for repeated fields
		if arr, ok := value.([]interface{}); ok {
			setRepeatedProperty(resources, targetPropId, arr, any)
			continue
		}

		// Set scalar value
		modifiedId := injectIndexOrKey(targetPropId, workSpace)
		instance, err := properties.PropertyOf(modifiedId, resources)
		if err != nil || instance == nil {
			continue
		}
		coerced := coerceJsonValue(value, instance, resources, workSpace)
		if coerced == nil {
			continue
		}
		instance.Set(any, coerced)
	}

	return nil
}

// getJsonValue walks a dot-separated path into a JSON map.
func getJsonValue(data map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	var current interface{} = data

	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current, ok = m[part]
		if !ok {
			return nil
		}
	}
	return current
}

// setRepeatedProperty sets values from a JSON array onto repeated protobuf fields.
func setRepeatedProperty(resources ifs.IResources, propertyId string, arr []interface{}, any interface{}) {
	for i, item := range arr {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		for key, val := range itemMap {
			fullId := fmt.Sprintf("%s<{2}%d>.%s", propertyId, i, key)
			fullId = injectIndexOrKey(fullId, nil)
			instance, err := properties.PropertyOf(fullId, resources)
			if err != nil || instance == nil {
				continue
			}
			coerced := coerceJsonValue(val, instance, resources, nil)
			instance.Set(any, coerced)
		}
	}
}

// coerceJsonValue converts a JSON value to match the target property type.
func coerceJsonValue(value interface{}, instance *properties.Property, resources ifs.IResources, workSpace map[string]interface{}) interface{} {
	node := instance.Node()
	if node == nil {
		return value
	}
	typeName := node.TypeName

	switch v := value.(type) {
	case float64:
		switch typeName {
		case "uint32":
			return uint32(v)
		case "int32":
			return int32(v)
		case "int64":
			return int64(v)
		case "uint64":
			return uint64(v)
		case "float32":
			return float32(v)
		case "string":
			return strconv.FormatFloat(v, 'f', -1, 64)
		}
	case string:
		switch typeName {
		case "uint32":
			if n, err := strconv.ParseUint(v, 10, 32); err == nil {
				return uint32(n)
			}
		case "int32":
			if n, err := strconv.ParseInt(v, 10, 32); err == nil {
				return int32(n)
			}
		case "float64":
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return f
			}
		}
	case bool:
		if typeName == "string" {
			return strconv.FormatBool(v)
		}
	}

	// Fallback: use the existing coerceValue for non-basic types
	return coerceValue(resources, value, instance, workSpace)
}
