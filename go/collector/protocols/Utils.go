package protocols


func SetValue(row, col int32, colName string, value []byte, tbl *types.CTable) {
	if tbl == nil {
		return
	}
	if tbl.Rows == nil {
		tbl.Rows = make(map[int32]*types.CRow)
	}
	rowData, ok := tbl.Rows[row]
	if !ok {
		rowData = &types.CRow{}
		rowData.Data = make(map[int32][]byte)
		tbl.Rows[row] = rowData
	}
	rowData.Data[col] = value
	if value != nil && tbl.Columns[col] == "" {
		tbl.Columns[col] = colName
	}
}

func Keys(m *types.CMap) []string {
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
