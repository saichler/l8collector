package k8sclient

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
)

const (
	ResultMap   = "map"
	ResultTable = "table"
	ModeGet     = "get"
	ModeList    = "list"
)

// CacheSpec describes how a job should read from the collector cache.
type CacheSpec struct {
	Result           string   `json:"result"`
	Mode             string   `json:"mode"`
	GVR              string   `json:"gvr"`
	Operations       []string `json:"operations"`
	Namespace        string   `json:"namespace"`
	NamespaceFromArg string   `json:"namespaceFromArg"`
	Name             string   `json:"name"`
	NameFromArg      string   `json:"nameFromArg"`
	Selector         string   `json:"selector"`
	SelectorFromArg  string   `json:"selectorFromArg"`
	Fields           []string `json:"fields"`
	Columns          []string `json:"columns"`
	ColumnNames      []string `json:"columnNames"`
}

func ParseCacheSpec(raw string, poll *l8tpollaris.L8Poll) (*CacheSpec, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("poll.What is empty")
	}
	spec := &CacheSpec{}
	err := json.Unmarshal([]byte(raw), spec)
	if err != nil {
		return nil, err
	}
	spec.applyDefaults(poll)
	if spec.GVR == "" {
		return nil, errors.New("cache spec gvr is empty")
	}
	if spec.Mode == ModeGet && spec.Name == "" && spec.NameFromArg == "" {
		return nil, errors.New("cache spec get requires name or nameFromArg")
	}
	return spec, nil
}

func (s *CacheSpec) applyDefaults(poll *l8tpollaris.L8Poll) {
	s.Result = strings.ToLower(strings.TrimSpace(s.Result))
	s.Mode = strings.ToLower(strings.TrimSpace(s.Mode))
	if s.Mode == "" {
		if s.Name != "" || s.NameFromArg != "" {
			s.Mode = ModeGet
		} else {
			s.Mode = ModeList
		}
	}
	if s.Result == "" {
		switch s.Mode {
		case ModeGet:
			s.Result = ResultMap
		default:
			s.Result = ResultTable
		}
	}
	if len(s.Columns) == 0 && len(s.Fields) > 0 {
		s.Columns = append([]string{}, s.Fields...)
	}
	if len(s.Fields) == 0 && len(s.Columns) > 0 {
		s.Fields = append([]string{}, s.Columns...)
	}
	if len(s.ColumnNames) == 0 && len(s.Columns) > 0 {
		s.ColumnNames = append([]string{}, s.Columns...)
	}
	if len(s.Fields) == 0 {
		s.Fields = []string{"metadata.name", "metadata.namespace", "metadata.uid"}
	}
	if len(s.Columns) == 0 {
		s.Columns = append([]string{}, s.Fields...)
	}
	if len(s.ColumnNames) == 0 {
		s.ColumnNames = append([]string{}, s.Columns...)
	}
	if len(s.Operations) == 0 {
		s.Operations = []string{"CREATE", "UPDATE", "DELETE"}
	}
	for i := range s.Operations {
		s.Operations[i] = strings.ToUpper(strings.TrimSpace(s.Operations[i]))
	}
	if poll != nil && s.Result == "" {
		if poll.Operation == l8tpollaris.L8C_Operation_L8C_Table {
			s.Result = ResultTable
		} else {
			s.Result = ResultMap
		}
	}
}

func resolveSpecValue(literal, argName string, args map[string]string) string {
	if argName != "" && args != nil {
		if value, ok := args[argName]; ok {
			return value
		}
	}
	return literal
}
