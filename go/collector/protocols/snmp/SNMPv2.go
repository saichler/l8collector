package snmp

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/saichler/l8collector/go/collector/protocols"
	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/types"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	strings2 "github.com/saichler/l8utils/go/utils/strings"
)

// normalizeOID converts ISO format OIDs to standard dotted decimal format
// Example: "iso.3.6.1.2.1.1.1.0" -> ".1.3.6.1.2.1.1.1.0"
func normalizeOID(oid string) string {
	if strings.HasPrefix(oid, "iso.") {
		return ".1." + oid[4:]
	}
	if !strings.HasPrefix(oid, ".") {
		return "." + oid
	}
	return oid
}

type SNMPv2Collector struct {
	resources ifs.IResources
	config    *types.Connection
	session   *SNMPSession
	connected bool
	pollOnce  bool
}

type SnmpPDU struct {
	Name  string
	Value interface{}
}

func (this *SNMPv2Collector) Protocol() types.Protocol {
	return types.Protocol_PSNMPV2
}

func (this *SNMPv2Collector) Init(conf *types.Connection, resources ifs.IResources) error {
	this.config = conf
	this.resources = resources
	return nil
}

func (this *SNMPv2Collector) Connect() error {
	if this == nil {
		return nil
	}
	// Create SNMP session using net-snmp library
	session, err := NewSNMPSession(this.config.Addr, this.config.ReadCommunity)
	if err != nil {
		return err
	}
	this.session = session
	this.connected = true
	return nil
}

func (this *SNMPv2Collector) Disconnect() error {
	if this.resources != nil && this.resources.Logger() != nil {
		this.resources.Logger().Info("SNMP Collector for ", this.config.Addr, " is closed.")
	}
	if this.session != nil {
		this.session.Close()
		this.session = nil
	}
	this.connected = false
	return nil
}

func (this *SNMPv2Collector) Exec(job *types.CJob) {
	this.pollOnce = true
	if this.resources != nil && this.resources.Logger() != nil {
		this.resources.Logger().Debug("Exec Job Start ", job.DeviceId, " ", job.PollarisName, ":", job.JobName)
	}
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
		if this.resources != nil && this.resources.Logger() != nil {
			this.resources.Logger().Error(strings2.New("SNMP:", err.Error()).String())
		}
		return
	}

	if poll.Operation == types.Operation_OMap {
		this.walk(job, poll, true)
	} else if poll.Operation == types.Operation_OTable {
		this.table(job, poll)
	}
	if this.resources != nil && this.resources.Logger() != nil {
		this.resources.Logger().Debug("Exec Job End  ", job.DeviceId, " ", job.PollarisName, ":", job.JobName)
	}
}

func (this *SNMPv2Collector) walk(job *types.CJob, poll *types.Poll, encodeMap bool) *types.CMap {
	// Add timeout wrapper for SNMP walk to prevent hanging on invalid OIDs
	timeout := time.Duration(this.config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second // Default 10 second timeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var pdus []SnmpPDU
	var e error

	done := make(chan bool)
	go func() {
		pdus, e = this.snmpWalk(poll.What)
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
			if this.resources != nil && this.resources.Logger() != nil {
				this.resources.Logger().Error("Object Value Error: ", err.Error())
			}
		}
		normalizedOID := normalizeOID(pdu.Name)
		m.Data[normalizedOID] = enc.Data()
	}
	if encodeMap {
		enc := object.NewEncode()
		err := enc.Add(m)
		if err != nil {
			if this.resources != nil && this.resources.Logger() != nil {
				this.resources.Logger().Error("Object Table Error: ", err)
			}
		}
		job.Result = enc.Data()
	}
	return m
}

func (this *SNMPv2Collector) snmpWalk(oid string) ([]SnmpPDU, error) {
	if this.session == nil {
		return nil, errors.New("SNMP session is not initialized")
	}

	// Use the net-snmp library session to perform the walk
	pdus, err := this.session.Walk(oid)
	if err != nil {
		return nil, err
	}

	return pdus, nil
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
		if this.resources != nil && this.resources.Logger() != nil {
			this.resources.Logger().Error("Object Table Error: ", err)
		}
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
