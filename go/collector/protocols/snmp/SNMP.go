package snmp

import (
	"github.com/gosnmp/gosnmp"
	"github.com/saichler/l8collector/go/collector/protocols"
	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/types"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	strings2 "github.com/saichler/l8utils/go/utils/strings"
	"strconv"
	"time"
)

type SNMPCollector struct {
	resources ifs.IResources
	config    *types.Connection
	agent     *gosnmp.GoSNMP
	connected bool
}

func (this *SNMPCollector) Protocol() types.Protocol {
	return types.Protocol_SNMPV2
}

func (this *SNMPCollector) Init(conf *types.Connection, resources ifs.IResources) error {
	this.config = conf
	this.resources = resources
	this.agent = &gosnmp.GoSNMP{}
	this.agent.Version = gosnmp.Version2c
	this.agent.Timeout = time.Second * time.Duration(this.config.Timeout)
	this.agent.Target = this.config.Addr
	this.agent.Port = uint16(this.config.Port)
	this.agent.Community = this.config.ReadCommunity
	this.agent.Retries = 1
	return nil
}

func (this *SNMPCollector) Connect() error {
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

func (this *SNMPCollector) Disconnect() error {
	this.resources.Logger().Info("SNMP Collector for ", this.config.Addr, " is closed.")
	this.agent = nil
	this.connected = false
	return nil
}

func (this *SNMPCollector) Exec(job *types.Job) {
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

	if poll.Operation == types.Operation_Map {
		this.walk(job, poll, true)
	} else if poll.Operation == types.Operation_Table {
		this.table(job, poll)
	}
	this.resources.Logger().Info("Exec Job End ", job.DeviceId, " ", job.PollarisName, ":", job.JobName)
}

func (this *SNMPCollector) walk(job *types.Job, poll *types.Poll, encodeMap bool) *types.CMap {
	if job.Timeout != 0 {
		this.agent.Timeout = time.Second * time.Duration(job.Timeout)
		defer func() { this.agent.Timeout = time.Second * time.Duration(this.config.Timeout) }()
	}
	pdus, e := this.agent.WalkAll(poll.What)
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

func (this *SNMPCollector) table(job *types.Job, poll *types.Poll) {
	m := this.walk(job, poll, false)
	if job.Error != "" {
		return
	}
	tbl := &types.CTable{}
	var lastRowIndex int32 = -1
	keys := protocols.Keys(m)
	var col int32 = 0
	for _, key := range keys {
		rowIndex, colIndex := getRowAndColName(key)
		if rowIndex > lastRowIndex {
			lastRowIndex = rowIndex
		}
		protocols.SetValue(rowIndex, col, colIndex, m.Data[key], tbl)
	}

	enc := object.NewEncode()
	err := enc.Add(tbl)
	if err != nil {
		this.resources.Logger().Error("Object Table Error: ", err)
		return
	}
	job.Result = enc.Data()
}
