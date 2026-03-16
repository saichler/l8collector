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
	"reflect"
	"strings"

	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8reflect/go/reflect/properties"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	strings2 "github.com/saichler/l8utils/go/utils/strings"
)

// CTableToMapProperty is a parsing rule that transforms a CTable structure into property values
// on a target object. It iterates through table rows, using key columns to generate PropertyId
// paths and setting column values to corresponding object properties.
type CTableToMapProperty struct{}

// Name returns the rule identifier "CTableToMapProperty".
func (this *CTableToMapProperty) Name() string {
	return "CTableToMapProperty"
}

// ParamNames returns the required parameter names for this rule.
func (this *CTableToMapProperty) ParamNames() []string {
	return []string{""}
}

// Parse executes the CTableToMapProperty rule, mapping table data to object properties.
func (this *CTableToMapProperty) Parse(resources ifs.IResources, workSpace map[string]interface{}, params map[string]*l8tpollaris.L8PParameter, any interface{}, pollWhat string) error {
	table, ok := workSpace[Output].(*l8tpollaris.CTable)
	if !ok {
		outVal := workSpace[Output]
		actualType := "nil"
		if outVal != nil {
			actualType = reflect.TypeOf(outVal).String()
		}
		return errors.New("Workspace had an invalid output object, expected *CTable but got " + actualType)
	}

	keyColumns, e := getIntArrInput(workSpace, KeyColumn)
	if e != nil {
		return e
	}

	propertyId := workSpace[PropertyId].(string)
	toString := strings2.New()
	toString.TypesPrefix = true

	for _, row := range table.Rows {
		pid := strings2.New(propertyId)
		recOK := true
		for i := 0; i < len(table.Columns); i++ {
			if i == 0 {
				pid.Add("<")
				for _, j := range keyColumns {
					val := getValue(row.Data[int32(j)], resources)
					if val == nil {
						recOK = false
					}
					pid.Add(toString.ToString(reflect.ValueOf(val)))
				}
				pid.Add(">.")
				if !recOK {
					break
				}
			}

			key := strings2.New(pid.String())
			attrName := getAttributeNameFromColumn(table.Columns[int32(i)])
			key.Add(attrName)
			keyString := key.String()
			prop, err := properties.PropertyOf(keyString, resources)
			if err != nil {
				resources.Logger().Error(err.Error())
				continue
			}

			val := getValue(row.Data[int32(i)], resources)
			_, _, err = prop.Set(any, val)
			if err != nil {
				resources.Logger().Error(err.Error())
				continue
			}
		}
	}
	return nil
}

func getAttributeNameFromColumn(value interface{}) string {
	colName := strings.TrimSpace(value.(string))
	colName = strings.ToLower(colName)
	colName = removeChar(colName, "-")
	colName = removeChar(colName, " ")
	colName = removeChar(colName, "(")
	colName = removeChar(colName, ")")
	return colName
}

func removeChar(colName, c string) string {
	index := strings.LastIndex(colName, c)
	if index == -1 {
		return colName
	}
	result := strings2.New(colName[0:index], colName[index+1:]).String()
	return removeChar(result, c)
}

func getValue(data []byte, resources ifs.IResources) interface{} {
	if len(data) == 0 {
		return nil
	}
	obj := object.NewDecode(data, 0, resources.Registry())
	val, _ := obj.Get()
	return val
}
