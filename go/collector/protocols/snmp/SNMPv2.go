package snmp

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gosnmp/gosnmp"
	"github.com/saichler/l8collector/go/collector/protocols"
	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/types"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	strings2 "github.com/saichler/l8utils/go/utils/strings"
)

var mtx = &sync.Mutex{}

type SNMPv2Collector struct {
	resources ifs.IResources
	config    *types.Connection
	agent     *gosnmp.GoSNMP
	connected bool
	pollOnce  bool
}

func (this *SNMPv2Collector) Protocol() types.Protocol {
	return types.Protocol_PSNMPV2
}

func (this *SNMPv2Collector) Init(conf *types.Connection, resources ifs.IResources) error {
	this.config = conf
	this.resources = resources
	this.agent = &gosnmp.GoSNMP{}
	this.agent.Version = gosnmp.Version2c
	this.agent.Timeout = time.Second * time.Duration(5)
	this.agent.Target = this.config.Addr
	this.agent.Port = uint16(this.config.Port)
	this.agent.Community = this.config.ReadCommunity
	this.agent.Retries = 3
	return nil
}

func (this *SNMPv2Collector) Connect() error {
	if this == nil || this.agent == nil {
		return nil
	}
	err := this.agent.Connect()
	if err != nil {
		return err
	}
	this.connected = true
	return nil
}

func (this *SNMPv2Collector) Disconnect() error {
	this.resources.Logger().Info("SNMP Collector for ", this.config.Addr, " is closed.")
	this.agent = nil
	this.connected = false
	return nil
}

func (this *SNMPv2Collector) Exec(job *types.CJob) {
	this.pollOnce = true
	this.resources.Logger().Info("Exec Job Start ", job.DeviceId, " ", job.PollarisName, ":", job.JobName)
	if !this.connected {
		err := this.Connect()
		if err != nil {
			job.Error = err.Error()
			return
		}
	}
	poll, err := pollaris.Poll(job.PollarisName, job.JobName, this.resources)
	if err != nil {
		this.resources.Logger().Error("SNMP:" + err.Error())
		return
	}

	if poll.Operation == types.Operation_OMap {
		this.walk(job, poll, true)
	} else if poll.Operation == types.Operation_OTable {
		this.table(job, poll)
	}
	this.resources.Logger().Info("Exec Job End ", job.DeviceId, " ", job.PollarisName, ":", job.JobName)
}

func (this *SNMPv2Collector) walk(job *types.CJob, poll *types.Poll, encodeMap bool) *types.CMap {
	if job.Timeout != 0 {
		this.agent.Timeout = time.Second * time.Duration(job.Timeout)
		defer func() { this.agent.Timeout = time.Second * time.Duration(this.config.Timeout) }()
	}
	// For Entity MIB, add strict timeout to prevent hanging
	var pdus []gosnmp.SnmpPDU
	var e error

	mtx.Lock()
	this.resources.Logger().Error("Before polling ", poll.What)
	pdus, e = this.agent.WalkAll(poll.What)
	this.resources.Logger().Error("After polling ", poll.What)
	mtx.Unlock()

	if e != nil {
		job.Error = strings2.New("SNMP Error Walk Host:", this.config.Addr, "/",
			strconv.Itoa(int(this.config.Port)), " Oid:", poll.What, e.Error()).String()
		return nil
	}
	m := &types.CMap{}
	m.Data = make(map[string][]byte)
	for _, pdu := range pdus {
		enc := object.NewEncode()
		err := enc.Add(pdu.Value)
		if err != nil {
			this.resources.Logger().Error("Object Value Error: ", err.Error())
		}
		m.Data[pdu.Name] = enc.Data()
	}
	if encodeMap {
		enc := object.NewEncode()
		err := enc.Add(m)
		if err != nil {
			this.resources.Logger().Error("Object Table Error: ", err)
		}
		job.Result = enc.Data()
	}
	return m
}

func (this *SNMPv2Collector) table(job *types.CJob, poll *types.Poll) {
	m := this.walk(job, poll, false)
	if job.Error != "" {
		return
	}
	tbl := &types.CTable{Rows: make(map[int32]*types.CRow), Columns: make(map[int32]string)}
	var lastRowIndex int32 = -1
	keys := protocols.Keys(m)

	for _, key := range keys {
		rowIndex, colIndex := getRowAndColName(key)
		if rowIndex > lastRowIndex {
			lastRowIndex = rowIndex
		}
		colInt, _ := strconv.Atoi(colIndex)
		protocols.SetValue(rowIndex, int32(colInt), colIndex, m.Data[key], tbl)
	}

	enc := object.NewEncode()
	err := enc.Add(tbl)
	if err != nil {
		this.resources.Logger().Error("Object Table Error: ", err)
		return
	}
	job.Result = enc.Data()
}

func (this *SNMPv2Collector) Online() bool {
	return this.connected || !this.pollOnce
}

func getRowAndColName(oid string) (int32, string) {
	index := strings.LastIndex(oid, ".")
	if index != -1 {
		row, _ := strconv.Atoi(oid[index+1:])
		suboid := oid[0:index]
		index = strings.LastIndex(suboid, ".")
		if index != -1 {
			col := suboid[index+1:]
			return int32(row), col
		}
	}
	return -1, ""
}
