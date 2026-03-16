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
	"strings"

	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8reflect/go/reflect/properties"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
)

// RestGpuParse is a parsing rule that extracts per-GPU fields from a REST API
// JSON response containing a GPU array. It uses the "pci_bus_id" field from each
// GPU object as the map key (same as SnmpGpuTable and SshNvidiaSmiParse).
//
// Parameters:
//   - "array_path": dot-path to the GPU array in the JSON (e.g., "devices")
//   - "mapping": comma-separated "jsonField:propertyName" pairs
//     Example: "compute_capability:computecapability,memory_total_mib:vramtotalmib"
//   - "key_field": JSON field name for the map key (default "pci_bus_id")
type RestGpuParse struct{}

func (this *RestGpuParse) Name() string {
	return "RestGpuParse"
}

func (this *RestGpuParse) ParamNames() []string {
	return []string{"array_path", "mapping"}
}

func (this *RestGpuParse) Parse(resources ifs.IResources, workSpace map[string]interface{},
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
			return errors.New("RestGpuParse: CMap has no 'json' key")
		}
		dec := object.NewDecode(jsonBytes, 0, resources.Registry())
		val, err := dec.Get()
		if err != nil {
			return errors.New("RestGpuParse: failed to decode json from CMap: " + err.Error())
		}
		jsonStr, _ = val.(string)
	} else if s, ok := input.(string); ok {
		jsonStr = s
	} else {
		return errors.New("RestGpuParse: unsupported input type: " + fmt.Sprintf("%T", input))
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &jsonData); err != nil {
		return errors.New("RestGpuParse: failed to parse JSON: " + err.Error())
	}

	arrayPathParam := params["array_path"]
	mappingParam := params["mapping"]
	if arrayPathParam == nil || mappingParam == nil {
		return errors.New("RestGpuParse: missing 'array_path' or 'mapping' parameter")
	}

	// Get the GPU array from JSON
	arrValue := getJsonValue(jsonData, arrayPathParam.Value)
	if arrValue == nil {
		return nil
	}
	gpuArray, ok := arrValue.([]interface{})
	if !ok {
		return errors.New("RestGpuParse: value at '" + arrayPathParam.Value + "' is not an array")
	}

	// Key field for map key (default pci_bus_id)
	keyField := "pci_bus_id"
	if kp := params["key_field"]; kp != nil && kp.Value != "" {
		keyField = kp.Value
	}

	// Parse mapping: "jsonField:propertyName,..."
	type fieldMapping struct {
		jsonField    string
		propertyName string
	}
	mappings := make([]fieldMapping, 0)
	for _, entry := range strings.Split(mappingParam.Value, ",") {
		parts := strings.SplitN(strings.TrimSpace(entry), ":", 2)
		if len(parts) != 2 {
			continue
		}
		mappings = append(mappings, fieldMapping{
			jsonField:    strings.TrimSpace(parts[0]),
			propertyName: strings.TrimSpace(parts[1]),
		})
	}

	propertyId := ""
	if pid, ok := workSpace[PropertyId]; ok {
		propertyId, _ = pid.(string)
	}

	// Iterate GPU array, use pci_bus_id as map key
	for _, item := range gpuArray {
		gpuMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		mapKey, ok := gpuMap[keyField].(string)
		if !ok || mapKey == "" {
			continue
		}
		mapKey = strings.TrimSpace(mapKey)

		// Set the PCI Bus ID as a property on the GPU instance
		setGpuProperty(resources, propertyId, mapKey, "pcibusid", mapKey, any)

		for _, m := range mappings {
			val, exists := gpuMap[m.jsonField]
			if !exists || val == nil {
				continue
			}
			fullId := fmt.Sprintf("%s<{24}%s>.%s", propertyId, mapKey, m.propertyName)
			instance, err := properties.PropertyOf(fullId, resources)
			if err != nil || instance == nil {
				continue
			}
			coerced := coerceJsonValue(val, instance, resources, workSpace)
			instance.Set(any, coerced)
		}
	}

	return nil
}
