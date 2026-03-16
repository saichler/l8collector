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

	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8reflect/go/reflect/properties"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
)

// Set is a parsing rule that directly sets a value from input to a target property.
// It supports PropertyId path injection for nested collections and handles type-safe value assignment.
// Parameters: "from" (source field in the input data).
type Set struct{}

// Name returns the rule identifier "Set".
func (this *Set) Name() string {
	return "Set"
}

// ParamNames returns the required parameter names for this rule.
func (this *Set) ParamNames() []string {
	return []string{}
}

// Parse executes the Set rule logic, extracting a value and setting it to the target property.
func (this *Set) Parse(resources ifs.IResources, workSpace map[string]interface{}, params map[string]*l8tpollaris.L8PParameter, any interface{}, pollWhat string) error {
	input := workSpace[Input]
	_propertyId := workSpace[PropertyId]
	propertyId := _propertyId.(string)

	if input == nil {
		return resources.Logger().Error("nil input for job")
	}

	value, _, err := GetValueInput(resources, input, params, pollWhat)
	if err != nil || value == nil {
		// Missing/blank OID data is expected for some devices — skip gracefully
		return nil
	}

	// Skip SNMP error strings gracefully - device doesn't support this OID
	if strVal, ok := value.(string); ok {
		if isSnmpErrorString(strVal) {
			return nil
		}
	}

	if _propertyId != nil {
		// Inject slice index or map key into PropertyId before creating property instance
		modifiedPropertyId := injectIndexOrKey(propertyId, workSpace)

		instance, err := properties.PropertyOf(modifiedPropertyId, resources)
		if err != nil {
			return resources.Logger().Error("error parsing instance path", err.Error())
		}
		if instance != nil {
			value = coerceValue(resources, value, instance, workSpace)
			_, _, err = instance.Set(any, value)
			if err != nil {
				return resources.Logger().Error("error setting property value:", err.Error())
			}
		}
	}
	workSpace[Output] = value
	return nil
}

// coerceValue converts the input value to match the target property's type when
// a direct assignment would fail. Handles SNMP int64→bool (TruthValue) and
// int64→*L8TimeSeriesPoint conversions.
func coerceValue(resources ifs.IResources, value interface{}, instance *properties.Property, workSpace map[string]interface{}) interface{} {
	node := instance.Node()
	if node == nil {
		return value
	}
	typeName := node.TypeName

	valueKind := reflect.TypeOf(value).Kind()

	// int64 → bool (SNMP TruthValue: 1=true, 2=false; ifAdminStatus: 1=up, 2=down)
	if typeName == "bool" && (valueKind == reflect.Int64 || valueKind == reflect.Int) {
		return reflect.ValueOf(value).Int() == 1
	}

	// int64/uint64/float64 → *L8TimeSeriesPoint (wrap numeric into time series point)
	if typeName == "L8TimeSeriesPoint" {
		var floatVal float64
		switch valueKind {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			floatVal = float64(reflect.ValueOf(value).Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			floatVal = float64(reflect.ValueOf(value).Uint())
		case reflect.Float32, reflect.Float64:
			floatVal = reflect.ValueOf(value).Float()
		default:
			return value
		}
		var stamp int64
		if ended, ok := workSpace[JobEnded]; ok {
			if s, ok := ended.(int64); ok {
				stamp = s
			}
		}
		return &l8api.L8TimeSeriesPoint{Stamp: stamp, Value: floatVal}
	}

	// Numeric → string (e.g., SNMP uptime timeticks arriving as int64 for a string field)
	if typeName == "string" {
		switch valueKind {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return fmt.Sprintf("%d", reflect.ValueOf(value).Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return fmt.Sprintf("%d", reflect.ValueOf(value).Uint())
		case reflect.Float32, reflect.Float64:
			return fmt.Sprintf("%g", reflect.ValueOf(value).Float())
		}
	}

	// Numeric → numeric (e.g., SNMP int64 → uint32/int32/float64 protobuf fields)
	if valueKind == reflect.Int64 || valueKind == reflect.Int {
		intVal := reflect.ValueOf(value).Int()
		switch typeName {
		case "int32":
			return int32(intVal)
		case "uint32":
			return uint32(intVal)
		case "uint64":
			return uint64(intVal)
		case "float32":
			return float32(intVal)
		case "float64":
			return float64(intVal)
		}
	}

	if valueKind == reflect.Uint64 || valueKind == reflect.Uint {
		uintVal := reflect.ValueOf(value).Uint()
		switch typeName {
		case "int32":
			return int32(uintVal)
		case "int64":
			return int64(uintVal)
		case "uint32":
			return uint32(uintVal)
		case "float32":
			return float32(uintVal)
		case "float64":
			return float64(uintVal)
		}
	}

	if valueKind == reflect.Float64 || valueKind == reflect.Float32 {
		floatVal := reflect.ValueOf(value).Float()
		switch typeName {
		case "int32":
			return int32(floatVal)
		case "int64":
			return int64(floatVal)
		case "uint32":
			return uint32(floatVal)
		case "uint64":
			return uint64(floatVal)
		}
	}

	// For non-basic type names (e.g., protobuf enums like ComponentStatus, ModuleType),
	// resolve the underlying kind via the introspector and coerce accordingly.
	if !isBasicTypeName(typeName) {
		nodeKind := resources.Introspector().Kind(node)
		switch nodeKind {
		case reflect.Int32:
			// Protobuf enums have underlying kind int32
			switch valueKind {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				return int32(reflect.ValueOf(value).Int())
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				return int32(reflect.ValueOf(value).Uint())
			case reflect.Float32, reflect.Float64:
				return int32(reflect.ValueOf(value).Float())
			default:
				// String or other non-numeric value for an enum field — skip gracefully
				return int32(0)
			}
		}
	}

	// Check if the value type matches the node type
	valueType := reflect.TypeOf(value).String()
	if valueType != typeName {
		propertyId, _ := instance.PropertyId()
		resources.Logger().Debug("coerceValue type mismatch: property=", propertyId, ", nodeType=", typeName, ", valueType=", valueType, ", value=", value)
		return defaultValueForType(typeName)
	}

	return value
}

// isBasicTypeName returns true if the type name is a Go basic type.
func isBasicTypeName(typeName string) bool {
	switch typeName {
	case "string", "bool",
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64":
		return true
	}
	return false
}

// defaultValueForType returns the zero/default value for a given type name.
func defaultValueForType(typeName string) interface{} {
	switch typeName {
	case "string":
		return ""
	case "bool":
		return false
	case "int32":
		return int32(0)
	case "int64":
		return int64(0)
	case "uint32":
		return uint32(0)
	case "uint64":
		return uint64(0)
	case "float32":
		return float32(0)
	case "float64":
		return float64(0)
	default:
		return nil
	}
}
