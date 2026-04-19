package k8sclient

import (
	"fmt"
	"strings"

	"github.com/saichler/l8collector/go/collector/protocols"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8srlz/go/serialize/object"
)

func BuildCMap(obj *CachedObject, fields []string) (*l8tpollaris.CMap, error) {
	cmap := &l8tpollaris.CMap{Data: make(map[string][]byte)}
	if obj == nil {
		return cmap, nil
	}
	if len(fields) == 0 {
		fields = []string{"object"}
	}
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		var value interface{}
		switch field {
		case "object":
			value = obj.Object
		case "related":
			value = obj.Related
		default:
			resolved, _ := FieldValue(obj, field)
			value = resolved
		}
		enc := object.NewEncode()
		err := enc.Add(value)
		if err != nil {
			return nil, err
		}
		cmap.Data[field] = enc.Data()
	}
	return cmap, nil
}

func BuildCTable(items []*CachedObject, fields, columnNames []string) (*l8tpollaris.CTable, error) {
	if len(fields) == 0 {
		fields = []string{"metadata.name", "metadata.namespace", "metadata.uid"}
	}
	if len(columnNames) == 0 {
		columnNames = append([]string{}, fields...)
	}
	if len(fields) != len(columnNames) {
		return nil, fmt.Errorf("fields/columnNames length mismatch: %d != %d", len(fields), len(columnNames))
	}
	tbl := &l8tpollaris.CTable{
		Rows:    make(map[int32]*l8tpollaris.CRow),
		Columns: make(map[int32]string),
	}
	for rowIdx, item := range items {
		for colIdx, field := range fields {
			value, _ := FieldValue(item, field)
			enc := object.NewEncode()
			err := enc.Add(value)
			if err != nil {
				return nil, fmt.Errorf("encode %s: %w", field, err)
			}
			protocols.SetValue(int32(rowIdx), int32(colIdx), columnNames[colIdx], enc.Data(), tbl)
		}
	}
	return tbl, nil
}

func FieldValue(obj *CachedObject, field string) (interface{}, bool) {
	if obj == nil {
		return nil, false
	}
	switch field {
	case "gvr":
		return obj.GVR, true
	case "namespace", "metadata.namespace":
		return obj.Namespace, true
	case "name", "metadata.name":
		return obj.Name, true
	case "uid", "metadata.uid":
		return obj.UID, true
	case "resourceVersion", "metadata.resourceVersion":
		return obj.ResourceVersion, true
	case "operation":
		return obj.Operation, true
	case "observedAt":
		return obj.ObservedAt, true
	case "object":
		return obj.Object, true
	case "related":
		return obj.Related, true
	default:
		return nestedValue(obj.Object, strings.Split(field, "."))
	}
}

func nestedValue(value interface{}, path []string) (interface{}, bool) {
	current := value
	for _, part := range path {
		switch typed := current.(type) {
		case map[string]interface{}:
			next, ok := typed[part]
			if !ok {
				return nil, false
			}
			current = next
		default:
			return nil, false
		}
	}
	return current, true
}

func stringify(value interface{}) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}
