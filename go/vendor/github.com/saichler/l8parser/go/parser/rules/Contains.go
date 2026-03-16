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
	"strings"

	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8reflect/go/reflect/properties"
)

// Contains is a parsing rule that checks if input data contains a specified substring.
// If the substring is found (case-insensitive), it sets the output value to the target property.
// Parameters: "what" (substring to search for), "from" (source field), "output" (value to set if found).
type Contains struct{}

// Name returns the rule identifier "Contains".
func (this *Contains) Name() string {
	return "Contains"
}

// ParamNames returns the required parameter names for this rule.
func (this *Contains) ParamNames() []string {
	return []string{"what"}
}

// Parse executes the Contains rule logic, checking if input contains the "what" substring.
func (this *Contains) Parse(resources ifs.IResources, workSpace map[string]interface{}, params map[string]*l8tpollaris.L8PParameter, any interface{}, pollWhat string) error {
	input := workSpace[Input]
	what := params[What]
	output := params[Output]
	path := workSpace[PropertyId]

	if input == nil {
		return resources.Logger().Error("nil input for job")
	}
	if what == nil {
		return resources.Logger().Error("nil 'what' parameter")
	}
	if output == nil {
		return resources.Logger().Error("Nil 'output' parameter")
	}
	value, kind, err := GetValueInput(resources, input, params, pollWhat)
	if err != nil {
		return err
	}

	str, err := convertToString(value, kind)
	if err != nil {
		return err
	}
	ok := strings.Contains(strings.ToLower(str), what.Value)
	if ok {
		if path != nil {
			instance, _ := properties.PropertyOf(path.(string), resources)
			if instance != nil {
				_, _, err := instance.Set(any, output.Value)
				if err != nil {
					return err
				}
			}
		}
		workSpace[Output] = output.Value
	}
	return nil
}
