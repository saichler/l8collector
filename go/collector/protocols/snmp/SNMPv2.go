package snmp

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/alouca/gosnmp"
	"github.com/saichler/l8collector/go/collector/protocols"
	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/types"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	strings2 "github.com/saichler/l8utils/go/utils/strings"
)

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
	timeout := int64(5)
	if conf.Timeout > 0 {
		timeout = int64(conf.Timeout)
	}
	agent, err := gosnmp.NewGoSNMP(this.config.Addr, this.config.ReadCommunity, gosnmp.Version2c, timeout)
	if err != nil {
		return err
	}
	this.agent = agent
	return nil
}

func (this *SNMPv2Collector) Connect() error {
	if this == nil || this.agent == nil {
		return nil
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
	this.resources.Logger().Debug("Exec Job Start ", job.DeviceId, " ", job.PollarisName, ":", job.JobName)
	if !this.connected {
		err := this.Connect()
		if err != nil {
			job.Error = err.Error()
			job.Result = nil
			job.ErrorCount++
			return
		}
	}
	poll, err := pollaris.Poll(job.PollarisName, job.JobName, this.resources)
	if err != nil {
		this.resources.Logger().Error(strings2.New("SNMP:", err.Error()).String())
		return
	}

	if poll.Operation == types.Operation_OMap {
		this.walk(job, poll, true)
	} else if poll.Operation == types.Operation_OTable {
		this.table(job, poll)
	}
	this.resources.Logger().Debug("Exec Job End  ", job.DeviceId, " ", job.PollarisName, ":", job.JobName)
}

func (this *SNMPv2Collector) walk(job *types.CJob, poll *types.Poll, encodeMap bool) *types.CMap {
	// For Entity MIB, add strict timeout to prevent hanging
	var pdus []gosnmp.SnmpPDU
	var e error

	// Add timeout wrapper for SNMP walk to prevent hanging on invalid OIDs
	timeout := time.Duration(this.config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second // Default 10 second timeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	done := make(chan bool)
	go func() {
		pdus, e = this.agent.Walk(poll.What)
		done <- true
	}()

	select {
	case <-done:
		// Walk completed normally
	case <-ctx.Done():
		// Timeout occurred
		job.Error = strings2.New("SNMP Walk Timeout Host:", this.config.Addr, "/",
			int(this.config.Port), " Oid:", poll.What, " timeout after", timeout.String()).String()
		job.Result = nil
		job.ErrorCount++
		return nil
	}

	if e != nil {
		job.Error = strings2.New("SNMP Error Walk Host:", this.config.Addr, "/",
			int(this.config.Port), " Oid:", poll.What, e.Error()).String()
		job.Result = nil
		job.ErrorCount++
		return nil
	} else {
		job.ErrorCount = 0
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
