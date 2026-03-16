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
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	strings2 "github.com/saichler/l8utils/go/utils/strings"
)

// StringToCTable is a parsing rule that converts a multi-line string into a structured table (CTable).
// It parses tabular output (like CLI command output) by detecting columns from the header
// and extracting values from subsequent rows.
// Parameters: "columns" (expected number of columns), "keycolumn" (column indices for the key).
type StringToCTable struct{}

// Name returns the rule identifier "StringToCTable".
func (this *StringToCTable) Name() string {
	return "StringToCTable"
}

// ParamNames returns the required parameter names for this rule.
func (this *StringToCTable) ParamNames() []string {
	return []string{"columns", "keycolumn"}
}

// Parse executes the StringToCTable rule, converting a string input to a CTable structure.
func (this *StringToCTable) Parse(resources ifs.IResources, workSpace map[string]interface{}, params map[string]*l8tpollaris.L8PParameter, any interface{}, pollWhat string) error {
	input, ok := workSpace[Input].(string)
	if !ok {
		return nil
	}
	colmns, err := getIntInput(workSpace, Columns)
	if err != nil {
		return err
	}

	lines := strings.Split(input, "\n")
	table := &l8tpollaris.CTable{}
	table.Rows = make(map[int32]*l8tpollaris.CRow)
	for i, line := range lines {
		if table.Columns == nil {
			table.Columns = getColumns(line, colmns)
			if len(table.Columns) != colmns {
				return resources.Logger().Error("Number of columns mismatch, expected:", colmns, ", actual:", len(table.Columns))
			}
			continue
		}
		row := &l8tpollaris.CRow{}
		row.Data = getValues(line, table.Columns)
		table.Rows[int32(i)] = row
	}
	workSpace[Output] = table
	return nil
}

func getValues(line string, columns map[int32]string) map[int32][]byte {
	line = strings.TrimSpace(line)
	result := make(map[int32][]byte, 0)
	begin := 0
	for i := 0; i < len(columns); i++ {
		col := columns[int32(i)]
		if begin+len(col) > len(line) {
			result[int32(i)] = []byte{}
		} else {
			value := strings.TrimSpace(line[begin : begin+len(col)])
			obj := object.NewEncode()
			obj.Add(value)
			result[int32(i)] = obj.Data()
			begin = begin + len(col)
		}
	}
	return result
}

func getColumns(line string, expected int) map[int32]string {
	columns := make([]string, 0)
	begin := 0
	open := false

	for i := 0; i < len(line); i++ {
		if line[i] == ' ' && !open {
			open = true
		} else if open && line[i] != ' ' {
			open = false
			col := line[begin:i]
			columns = append(columns, col)
			begin = i
		}
	}

	if begin < len(line)-1 {
		col := line[begin:len(line)]
		columns = append(columns, col)
	}
	if len(columns) > expected {
		newColumns := make([]string, 0)
		for i := 0; i < len(columns); i++ {
			l := len(columns[i])
			t := len(strings.TrimSpace(columns[i]))
			if l == t+1 {
				newColumns = append(newColumns, strings2.New(columns[i], columns[i+1]).String())
				i++
			} else {
				newColumns = append(newColumns, columns[i])
			}
		}
		columns = newColumns
	}
	result := make(map[int32]string)
	for i, col := range columns {
		result[int32(i)] = col
	}
	return result
}
