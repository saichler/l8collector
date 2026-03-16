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
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8reflect/go/reflect/properties"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
)

// SnmpGpuTable is a parsing rule that transforms NVIDIA GPU SNMP table data
// (indexed OIDs: {base}.{metric_id}.{gpu_index}) into GpuDevice.Gpus map entries.
// It uses a two-pass approach: first pass collects PCI Bus IDs (OID suffix 4) per GPU index,
// second pass sets properties using PCI Bus ID as the map key.
//
// Parameters:
//   - "oid_base": base OID prefix (e.g., "1.3.6.1.4.1.53246.1.1.1.1")
//   - "mapping": comma-separated "oidSuffix:propertyName:type" triples
//     type is "set" for static or "ts" for time series
//     Example: "1:devicename:set,2:deviceuuid:set,5:gpuutilizationpercent:ts"
//   - "key_oid": OID suffix that contains the map key (PCI Bus ID), default 4
type SnmpGpuTable struct{}

// Name returns the rule identifier "SnmpGpuTable".
func (this *SnmpGpuTable) Name() string {
	return "SnmpGpuTable"
}

// ParamNames returns the required parameter names for this rule.
func (this *SnmpGpuTable) ParamNames() []string {
	return []string{"oid_base", "mapping"}
}

// gpuFieldMapping holds a parsed mapping entry.
type gpuFieldMapping struct {
	oidSuffix    int
	propertyName string
	isTimeSeries bool
}

// Parse executes the SnmpGpuTable rule logic.
// Pass 1: collect PCI Bus IDs (key OID suffix) per GPU index.
// Pass 2: set properties using PCI Bus ID as the map key.
func (this *SnmpGpuTable) Parse(resources ifs.IResources, workSpace map[string]interface{},
	params map[string]*l8tpollaris.L8PParameter, any interface{}, pollWhat string) error {

	input := workSpace[Input]
	_propertyId := workSpace[PropertyId]
	if input == nil || _propertyId == nil {
		return nil
	}

	propertyId := _propertyId.(string)

	cmap, ok := input.(*l8tpollaris.CMap)
	if !ok {
		resources.Logger().Error("SnmpGpuTable: input is not a CMap, got ", fmt.Sprintf("%T", input))
		return nil
	}
	if cmap == nil || len(cmap.Data) == 0 {
		resources.Logger().Error("SnmpGpuTable: CMap is empty for propertyId ", _propertyId)
		return nil
	}

	// Parse parameters
	oidBaseParam := params["oid_base"]
	mappingParam := params["mapping"]
	if oidBaseParam == nil || mappingParam == nil {
		return resources.Logger().Error("SnmpGpuTable: missing oid_base or mapping parameter")
	}

	oidBase := oidBaseParam.Value
	if !strings.HasPrefix(oidBase, ".") {
		oidBase = "." + oidBase
	}

	mappings := parseMappings(mappingParam.Value)
	if len(mappings) == 0 {
		return resources.Logger().Error("SnmpGpuTable: no valid mappings parsed from: ", mappingParam.Value)
	}

	suffixMap := make(map[int]*gpuFieldMapping)
	for i := range mappings {
		suffixMap[mappings[i].oidSuffix] = &mappings[i]
	}

	// Key OID suffix for map key (default 4 = pcibusid)
	keyOidSuffix := 4
	if kp := params["key_oid"]; kp != nil {
		if v, err := strconv.Atoi(kp.Value); err == nil {
			keyOidSuffix = v
		}
	}

	var stamp int64
	if ended, ok := workSpace[JobEnded]; ok {
		if s, ok := ended.(int64); ok {
			stamp = s
		}
	}

	// Parse all OID entries into (gpuIndex, metricId, value) tuples
	type oidEntry struct {
		gpuIndex int
		metricId int
		value    interface{}
	}
	entries := make([]oidEntry, 0)
	for oidKey, rawData := range cmap.Data {
		if len(rawData) == 0 {
			continue
		}
		if !strings.HasPrefix(oidKey, oidBase) {
			continue
		}
		suffix := oidKey[len(oidBase):]
		if strings.HasPrefix(suffix, ".") {
			suffix = suffix[1:]
		}
		parts := strings.SplitN(suffix, ".", 2)
		if len(parts) != 2 {
			continue
		}
		metricId, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		gpuIndex, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		enc := object.NewDecode(rawData, 0, resources.Registry())
		value, err := enc.Get()
		if err != nil || value == nil {
			continue
		}
		if strVal, ok := value.(string); ok {
			if isSnmpErrorString(strVal) {
				continue
			}
		}
		entries = append(entries, oidEntry{gpuIndex, metricId, value})
	}

	// Pass 1: collect PCI Bus IDs per GPU index
	gpuKeys := make(map[int]string)
	for _, e := range entries {
		if e.metricId == keyOidSuffix {
			if strVal, ok := e.value.(string); ok {
				gpuKeys[e.gpuIndex] = strings.TrimSpace(strVal)
			}
		}
	}

	// Pass 2: set properties using PCI Bus ID as map key
	for _, e := range entries {
		mapKey, hasKey := gpuKeys[e.gpuIndex]
		if !hasKey {
			mapKey = fmt.Sprintf("gpu-%d", e.gpuIndex)
		}

		// Set gpu_index
		gpuIndexPropId := fmt.Sprintf("%s<{24}%s>.gpuindex", propertyId, mapKey)
		if inst, err := properties.PropertyOf(gpuIndexPropId, resources); err == nil && inst != nil {
			inst.Set(any, uint32(e.gpuIndex))
		}

		mapping, exists := suffixMap[e.metricId]
		if !exists {
			continue
		}

		fullPropertyId := fmt.Sprintf("%s<{24}%s>.%s", propertyId, mapKey, mapping.propertyName)

		if mapping.isTimeSeries {
			floatVal, err := toFloat64(e.value)
			if err != nil {
				continue
			}
			point := &l8api.L8TimeSeriesPoint{Stamp: stamp, Value: floatVal}
			instance, err := properties.PropertyOf(fullPropertyId, resources)
			if err != nil || instance == nil {
				continue
			}
			_, _, err = instance.Set(any, point)
			if err != nil {
				resources.Logger().Error("SnmpGpuTable: error setting time series for GPU ", mapKey, ":", err.Error())
			}
		} else {
			instance, err := properties.PropertyOf(fullPropertyId, resources)
			if err != nil || instance == nil {
				continue
			}
			e.value = coerceValue(resources, e.value, instance, workSpace)
			_, _, err = instance.Set(any, e.value)
			if err != nil {
				resources.Logger().Error("SnmpGpuTable: error setting value for GPU ", mapKey, ":", err.Error())
			}
		}
	}

	return nil
}

// parseMappings parses a comma-separated mapping string into gpuFieldMapping entries.
// Format: "oidSuffix:propertyName:type,oidSuffix:propertyName:type,..."
// type is "set" for static or "ts" for time series.
func parseMappings(mappingStr string) []gpuFieldMapping {
	result := make([]gpuFieldMapping, 0)
	entries := strings.Split(mappingStr, ",")
	for _, entry := range entries {
		parts := strings.SplitN(strings.TrimSpace(entry), ":", 3)
		if len(parts) != 3 {
			continue
		}
		oidSuffix, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		result = append(result, gpuFieldMapping{
			oidSuffix:    oidSuffix,
			propertyName: parts[1],
			isTimeSeries: parts[2] == "ts",
		})
	}
	return result
}

// toFloat64 converts a value to float64 for time series points.
func toFloat64(value interface{}) (float64, error) {
	kind := reflect.TypeOf(value).Kind()
	switch kind {
	case reflect.Float32, reflect.Float64:
		return reflect.ValueOf(value).Float(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(reflect.ValueOf(value).Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(reflect.ValueOf(value).Uint()), nil
	case reflect.String:
		return strconv.ParseFloat(value.(string), 64)
	default:
		return 0, fmt.Errorf("unsupported type for float64: %s", kind.String())
	}
}
