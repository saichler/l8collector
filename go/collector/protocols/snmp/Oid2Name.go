package snmp

import (
	"strconv"
	"strings"
	"sync"
)

type OidToName struct {
	oid2name map[string]string
	mtx      *sync.Mutex
}

var Oid2Name = newOidToName()

func newOidToName() *OidToName {
	otn := &OidToName{}
	otn.oid2name = make(map[string]string)
	otn.mtx = &sync.Mutex{}
	otn.init()
	return otn
}

func (otn *OidToName) init() {
	otn.mtx.Lock()
	defer otn.mtx.Unlock()
	otn.oid2name[".1.3.6.1.2.1.2.2.1.2"] = "ifDescr"
}

func (otn *OidToName) Get(oid string) (string, bool) {
	otn.mtx.Lock()
	defer otn.mtx.Unlock()
	name := otn.oid2name[oid]
	if name == "" {
		return oid, false
	}
	return name, true
}

func getRowAndColName(oid string) (int32, string) {
	index := strings.LastIndex(oid, ".")
	if index != -1 {
		row, _ := strconv.Atoi(oid[index+1:])
		suboid := oid[0:index]
		index = strings.LastIndex(suboid, ".")
		if index != -1 {
			col := suboid[index+1:]
			name, ok := Oid2Name.Get(suboid)
			if ok {
				col = name
			}
			return int32(row), col
		}
	}
	return -1, ""
}
