package protocols

import (
	"github.com/saichler/l8pollaris/go/types/l8poll"
)

func SetValue(row, col int32, colName string, value []byte, tbl *l8poll.CTable) {
	if tbl == nil {
		return
	}
	if tbl.Rows == nil {
		tbl.Rows = make(map[int32]*l8poll.CRow)
	}
	rowData, ok := tbl.Rows[row]
	if !ok {
		rowData = &l8poll.CRow{}
		rowData.Data = make(map[int32][]byte)
		tbl.Rows[row] = rowData
	}
	rowData.Data[col] = value
	if value != nil && tbl.Columns[col] == "" {
		tbl.Columns[col] = colName
	}
}

func Keys(m *l8poll.CMap) []string {
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
