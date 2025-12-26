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

// Package protocols provides utility functions shared across all protocol
// collector implementations. It includes helper functions for working with
// CTable and CMap data structures used in collection results.
package protocols

import (
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
)

// SetValue sets a value in a CTable at the specified row and column position.
// This function handles lazy initialization of the table's row map and individual
// row data structures. It also sets the column name if not already defined.
//
// This is primarily used by the SNMP collector when building table results
// from SNMP walk operations.
//
// Parameters:
//   - row: The row index (0-based) where the value should be stored
//   - col: The column index (0-based) where the value should be stored
//   - colName: The name to assign to this column if not already set
//   - value: The byte slice value to store at this cell
//   - tbl: The target CTable structure (must not be nil)
//
// If tbl is nil, the function returns without error.
func SetValue(row, col int32, colName string, value []byte, tbl *l8tpollaris.CTable) {
	if tbl == nil {
		return
	}
	if tbl.Rows == nil {
		tbl.Rows = make(map[int32]*l8tpollaris.CRow)
	}
	rowData, ok := tbl.Rows[row]
	if !ok {
		rowData = &l8tpollaris.CRow{}
		rowData.Data = make(map[int32][]byte)
		tbl.Rows[row] = rowData
	}
	rowData.Data[col] = value
	if value != nil && tbl.Columns[col] == "" {
		tbl.Columns[col] = colName
	}
}

// Keys extracts all keys from a CMap and returns them as a string slice.
// The order of keys in the result is not guaranteed due to Go's map iteration.
//
// This function is used when iterating over SNMP walk results stored in a CMap,
// particularly for converting map-based results into table format.
//
// Parameters:
//   - m: The CMap to extract keys from
//
// Returns:
//   - A slice of all keys in the map, or an empty slice if the map is nil
func Keys(m *l8tpollaris.CMap) []string {
	if m == nil || m.Data == nil {
		return []string{}
	}
	result := make([]string, len(m.Data))
	i := 0
	for k, _ := range m.Data {
		result[i] = k
		i++
	}
	return result
}
