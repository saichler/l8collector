package snmp

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	wapsnmp "github.com/cdevr/WapSNMP"
	"github.com/saichler/l8collector/go/collector/protocols"
	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/types/l8poll"
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
	resources   ifs.IResources
	config      *l8poll.L8T_Connection
	session     *wapsnmp.WapSNMP
	connected   bool
	pollSuccess bool
}

type SnmpPDU struct {
	Name  string
	Value interface{}
}

func (this *SNMPv2Collector) Protocol() l8poll.L8C_Protocol {
	return l8poll.L8C_Protocol_L8P_PSNMPV2
}

func (this *SNMPv2Collector) Init(conf *l8poll.L8T_Connection, resources ifs.IResources) error {
	this.config = conf
	this.resources = resources
	return nil
}

func (this *SNMPv2Collector) Connect() error {
	if this == nil {
		return nil
	}

	// Create WapSNMP instance using the NewWapSNMP constructor
	target := this.config.Addr
	community := this.config.ReadCommunity
	version := wapsnmp.SNMPv2c
	timeout := time.Duration(this.config.Timeout) * time.Second

	// Default timeout if not specified
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	session, err := wapsnmp.NewWapSNMP(target, community, version, timeout, 3)
	if err != nil {
		return fmt.Errorf("failed to create SNMP session for %s: %v", target, err)
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
		if err := this.session.Close(); err != nil && this.resources != nil && this.resources.Logger() != nil {
			this.resources.Logger().Error("Error closing SNMP session: ", err.Error())
		}
		this.session = nil
	}
	this.connected = false
	return nil
}

func (this *SNMPv2Collector) Exec(job *l8poll.CJob) {
	if this.resources != nil && this.resources.Logger() != nil {
		this.resources.Logger().Debug("Exec Job Start ", job.TargetId, " ", job.PollarisName, ":", job.JobName)
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

	if poll.Operation == l8poll.L8C_Operation_L8C_Map {
		this.walk(job, poll, true)
	} else if poll.Operation == l8poll.L8C_Operation_L8C_Table {
		this.table(job, poll)
	}
	if this.resources != nil && this.resources.Logger() != nil {
		this.resources.Logger().Debug("Exec Job End  ", job.TargetId, " ", job.PollarisName, ":", job.JobName)
	}
}

func (this *SNMPv2Collector) walk(job *l8poll.CJob, poll *l8poll.L8Poll, encodeMap bool) *l8poll.CMap {
	// Add timeout wrapper for SNMP walk to prevent hanging on invalid OIDs
	timeout := time.Duration(this.config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second // Default 10 second timeout
	}

	var pdus []SnmpPDU
	var lastError error

	// Try once with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	var e error
	done := make(chan bool)

	go func() {
		pdus, e = this.snmpWalk(poll.What)
		done <- true
	}()

	select {
	case <-done:
		this.pollSuccess = true
		cancel()
		// Walk completed normally
		if e == nil {
			// Success
			lastError = nil
		} else {
			// Error occurred
			lastError = e
		}

	case <-ctx.Done():
		cancel()
		// Timeout occurred
		lastError = fmt.Errorf("timeout after %s", timeout.String())
	}

	// Handle errors
	if lastError != nil {
		if strings.Contains(lastError.Error(), "timeout") {
			// Timeout error
			job.Error = strings2.New("SNMP Walk Timeout. Host:",
				this.config.Addr, "/", int(this.config.Port), " Oid:", poll.What, " ",
				lastError.Error()).String()
		} else {
			// Other SNMP error
			job.Error = strings2.New("SNMP Error Walk Host:", this.config.Addr, "/",
				int(this.config.Port), " Oid:", poll.What, " ", lastError.Error()).String()
		}
		job.Result = nil
		job.ErrorCount++
		return nil
	} else {
		job.ErrorCount = 0
	}

	m := &l8poll.CMap{}
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
		return nil, fmt.Errorf("SNMP session is not initialized")
	}

	// Parse OID string to WapSNMP Oid format
	parsedOid, err := wapsnmp.ParseOid(oid)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OID %s: %v", oid, err)
	}

	// Perform SNMP walk using iterative GetNext calls only
	var pdus []SnmpPDU
	currentOid := parsedOid.Copy()

	for {
		nextOid, value, err := this.session.GetNext(currentOid)
		if err != nil {
			break // End of walk or error
		}

		// Check if we're still within the requested subtree
		if !nextOid.Within(parsedOid) {
			break // We've walked beyond the requested subtree
		}

		pdus = append(pdus, SnmpPDU{
			Name:  nextOid.String(),
			Value: value,
		})

		// Move to the next OID
		currentOid = *nextOid
	}

	if len(pdus) == 0 {
		return nil, fmt.Errorf("SNMP walk found no results for OID %s", oid)
	}

	return pdus, nil
}

func (this *SNMPv2Collector) table(job *l8poll.CJob, poll *l8poll.L8Poll) {
	m := this.walk(job, poll, false)
	if job.Error != "" {
		return
	}
	tbl := &l8poll.CTable{Rows: make(map[int32]*l8poll.CRow), Columns: make(map[int32]string)}
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
	return this.connected && this.pollSuccess
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
