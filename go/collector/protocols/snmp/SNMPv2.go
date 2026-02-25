/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
You may obtain a copy of the License at:

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package snmp provides SNMP v2c protocol collector implementation for the
// L8Collector service. It enables data collection from network devices using
// SNMP GET, GETNEXT, and WALK operations with community-based authentication.
package snmp

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	wapsnmp "github.com/cdevr/WapSNMP"
	"github.com/saichler/l8collector/go/collector/protocols"
	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	strings2 "github.com/saichler/l8utils/go/utils/strings"
)

// normalizeOID converts ISO format OIDs to standard dotted decimal format.
// This handles variations in OID representation from different SNMP implementations.
//
// Examples:
//   - "iso.3.6.1.2.1.1.1.0" -> ".1.3.6.1.2.1.1.1.0"
//   - "1.3.6.1.2.1.1.1.0" -> ".1.3.6.1.2.1.1.1.0"
//
// Parameters:
//   - oid: The OID string to normalize
//
// Returns:
//   - The normalized OID string with leading dot and numeric format
func normalizeOID(oid string) string {
	if strings.HasPrefix(oid, "iso.") {
		return ".1." + oid[4:]
	}
	if !strings.HasPrefix(oid, ".") {
		return "." + oid
	}
	return oid
}

// SNMPv2Collector implements the ProtocolCollector interface for SNMP v2c.
// It provides SNMP walk and table operations for collecting data from
// network devices using community-based authentication.
//
// Features:
//   - SNMP v2c protocol support with community string authentication
//   - Configurable timeout with automatic fallback to net-snmp
//   - SNMP walk operations returning map or table formats
//   - Enhanced timeout protection with context-based cancellation
//   - Automatic OID normalization for consistent result formatting
//
// The collector uses the WapSNMP library as the primary SNMP implementation
// with automatic fallback to net-snmp command-line tools on timeout.
type SNMPv2Collector struct {
	resources   ifs.IResources                // Layer8 resources for logging and security
	config      *l8tpollaris.L8PHostProtocol  // Host configuration with address and credentials
	session     *wapsnmp.WapSNMP              // WapSNMP session for SNMP operations
	connected   bool                          // Connection state flag
	pollSuccess bool                          // Flag indicating at least one successful poll
}

// SnmpPDU represents a single SNMP Protocol Data Unit containing an OID
// name and its associated value. Used for collecting walk results.
type SnmpPDU struct {
	Name  string      // The OID in dotted decimal notation
	Value interface{} // The value associated with this OID
}

// Protocol returns the protocol type identifier for SNMP v2c.
// This is used by the collector service to route jobs to the correct collector.
func (this *SNMPv2Collector) Protocol() l8tpollaris.L8PProtocol {
	return l8tpollaris.L8PProtocol_L8PPSNMPV2
}

// Init initializes the SNMP collector with the provided host configuration.
// It stores the configuration and resources for later use during Connect.
//
// Parameters:
//   - conf: Host protocol configuration containing address, port, and credential ID
//   - resources: Layer8 resources for accessing security credentials and logging
//
// Returns:
//   - Always returns nil (initialization cannot fail)
func (this *SNMPv2Collector) Init(conf *l8tpollaris.L8PHostProtocol, resources ifs.IResources) error {
	this.config = conf
	this.resources = resources
	return nil
}

// Connect establishes the SNMP session with the target device.
// It retrieves the community string from the security service and creates
// a WapSNMP session configured for SNMP v2c with the specified timeout.
//
// The default timeout is 60 seconds if not specified in the configuration.
//
// Returns:
//   - error if credential retrieval or session creation fails
func (this *SNMPv2Collector) Connect() error {
	if this == nil {
		return nil
	}

	// Create WapSNMP instance using the NewWapSNMP constructor
	target := this.config.Addr
	_, readCommunity, _, _, err := this.resources.Security().Credential(this.config.CredId, "snmp", this.resources)
	if err != nil {
		panic(err)
	}
	community := readCommunity
	version := wapsnmp.SNMPv2c
	timeout := time.Duration(this.config.Timeout) * time.Second

	// Default timeout if not specified
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	session, err := wapsnmp.NewWapSNMP(target, community, version, timeout, 3)
	if err != nil {
		return fmt.Errorf("failed to create SNMP session for %s: %v", target, err)
	}

	this.session = session
	this.connected = true
	return nil
}

// Disconnect closes the SNMP session and releases all resources.
// It logs the closure and handles any errors during session close.
//
// Returns:
//   - Always returns nil (cleanup is best-effort)
func (this *SNMPv2Collector) Disconnect() error {
	if this.resources != nil && this.resources.Logger() != nil {
		this.resources.Logger().Debug("SNMP Collector for ", this.config.Addr, " is closed.")
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

// Exec executes an SNMP collection job against the target device.
// The operation type (Map or Table) is determined from the pollaris configuration.
// The method automatically establishes a connection if not already connected.
//
// Supported operations:
//   - L8C_Map: Performs SNMP walk and returns results as a CMap
//   - L8C_Table: Performs SNMP walk and structures results as a CTable
//
// Parameters:
//   - job: The collection job containing pollaris reference and result storage
func (this *SNMPv2Collector) Exec(job *l8tpollaris.CJob) {
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

	if poll.Operation == l8tpollaris.L8C_Operation_L8C_Get {
		this.get(job, poll)
	} else if poll.Operation == l8tpollaris.L8C_Operation_L8C_Map {
		this.walk(job, poll, true)
	} else if poll.Operation == l8tpollaris.L8C_Operation_L8C_Table {
		this.table(job, poll)
	}
	if this.resources != nil && this.resources.Logger() != nil {
		this.resources.Logger().Debug("Exec Job End  ", job.TargetId, " ", job.PollarisName, ":", job.JobName)
	}
}

// get performs an SNMP GET operation for a single OID.
// Unlike walk, which traverses an entire subtree using GetNext, get retrieves
// the value of a specific OID directly. The result is returned as an encoded
// CMap with a single OID->value entry.
//
// The method uses the same timeout and fallback strategy as walk:
//  1. Attempts GET using WapSNMP library with a timeout context
//  2. Falls back to net-snmp snmpget if timeout occurs
//
// Parameters:
//   - job: The collection job for storing results and errors
//   - poll: The poll configuration containing the OID to get
func (this *SNMPv2Collector) get(job *l8tpollaris.CJob, poll *l8tpollaris.L8Poll) {
	timeout := time.Duration(this.config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	var pdu *SnmpPDU
	var lastError error

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	done := make(chan bool)

	go func() {
		p, e := this.snmpGet(poll.What)
		if e == nil {
			pdu = p
		} else {
			lastError = e
		}
		done <- true
	}()

	select {
	case <-done:
		this.pollSuccess = true
		cancel()
	case <-ctx.Done():
		cancel()
		if this.resources != nil && this.resources.Logger() != nil {
			this.resources.Logger().Debug("SNMP GET timeout, trying net-snmp fallback for OID: ", poll.What)
		}

		netSnmp := NewNetSNMPCollector(this.config, this.resources)
		fallbackPdu, fallbackErr := netSnmp.snmpGet(poll.What)

		if fallbackErr == nil {
			pdu = fallbackPdu
			lastError = nil
		} else {
			lastError = fmt.Errorf("timeout after %s, net-snmp fallback also failed: %v", timeout.String(), fallbackErr)
		}
	}

	if lastError != nil {
		job.Error = strings2.New("SNMP Get Error Host:", this.config.Addr, "/",
			int(this.config.Port), " Oid:", poll.What, " ", lastError.Error()).String()
		job.Result = nil
		job.ErrorCount++
		return
	}
	job.ErrorCount = 0

	m := &l8tpollaris.CMap{}
	m.Data = make(map[string][]byte)

	enc := object.NewEncode()
	err := enc.Add(pdu.Value)
	if err != nil {
		if this.resources != nil && this.resources.Logger() != nil {
			this.resources.Logger().Error("Object Value Error: ", err.Error())
		}
	}
	normalizedOID := normalizeOID(pdu.Name)
	m.Data[normalizedOID] = enc.Data()

	encMap := object.NewEncode()
	err = encMap.Add(m)
	if err != nil {
		if this.resources != nil && this.resources.Logger() != nil {
			this.resources.Logger().Error("Object Map Error: ", err)
		}
	}
	job.Result = encMap.Data()
}

// snmpGet performs a single SNMP GET using WapSNMP's Get operation.
//
// Parameters:
//   - oid: The OID to get (e.g., ".1.3.6.1.2.1.1.1.0")
//
// Returns:
//   - SnmpPDU containing the OID and its value
//   - error if session is not initialized or GET fails
func (this *SNMPv2Collector) snmpGet(oid string) (*SnmpPDU, error) {
	if this.session == nil {
		return nil, fmt.Errorf("SNMP session is not initialized")
	}

	parsedOid, err := wapsnmp.ParseOid(oid)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OID %s: %v", oid, err)
	}

	value, err := this.session.Get(parsedOid)
	if err != nil {
		return nil, fmt.Errorf("SNMP GET failed for OID %s: %v", oid, err)
	}

	return &SnmpPDU{
		Name:  oid,
		Value: value,
	}, nil
}

// walk performs an SNMP walk operation starting from the specified OID.
// It implements timeout protection with automatic fallback to net-snmp
// command-line tools if the WapSNMP library times out.
//
// The walk process:
//  1. Creates a timeout context based on configuration
//  2. Attempts walk using WapSNMP library
//  3. Falls back to net-snmp if timeout occurs
//  4. Normalizes OIDs and encodes results
//
// Parameters:
//   - job: The collection job for storing results and errors
//   - poll: The poll configuration containing the base OID
//   - encodeMap: Whether to encode the result map for storage
//
// Returns:
//   - CMap containing OID->value mappings, or nil on error
func (this *SNMPv2Collector) walk(job *l8tpollaris.CJob, poll *l8tpollaris.L8Poll, encodeMap bool) *l8tpollaris.CMap {
	// Add timeout wrapper for SNMP walk to prevent hanging on invalid OIDs
	timeout := time.Duration(this.config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second // Default 60 second timeout
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
		// Timeout occurred - try net-snmp fallback
		if this.resources != nil && this.resources.Logger() != nil {
			this.resources.Logger().Debug("SNMP timeout, trying net-snmp fallback for OID: ", poll.What)
		}

		netSnmp := NewNetSNMPCollector(this.config, this.resources)
		// Remove leading "." for net-snmp compatibility
		fallbackPdus, fallbackErr := netSnmp.snmpWalk(poll.What)

		if fallbackErr == nil {
			pdus = fallbackPdus
			lastError = nil
			if this.resources != nil && this.resources.Logger() != nil {
				this.resources.Logger().Debug("net-snmp fallback succeeded for OID: ", poll.What)
			}
		} else {
			lastError = fmt.Errorf("timeout after %s, net-snmp fallback also failed: %v", timeout.String(), fallbackErr)
			if this.resources != nil && this.resources.Logger() != nil {
				this.resources.Logger().Warning("net-snmp fallback failed for OID: ", poll.What, " error: ",
					job.TargetId, " ",
					os.Getenv("HOSTNAME"), " ",
					fallbackErr.Error())
			}
		}
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

	m := &l8tpollaris.CMap{}
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

// snmpWalk performs the actual SNMP walk using WapSNMP's GetNext operations.
// It iteratively retrieves OIDs within the specified subtree until it reaches
// an OID outside the subtree or encounters an error.
//
// Parameters:
//   - oid: The base OID to walk from (e.g., ".1.3.6.1.2.1.2.2.1")
//
// Returns:
//   - Slice of SnmpPDU containing all OID-value pairs found
//   - error if session is not initialized or walk finds no results
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

// table performs an SNMP walk and structures the results as a table (CTable).
// It extracts row and column indices from the OIDs and organizes the data
// into a row/column structure suitable for tabular MIB data.
//
// The method parses OIDs to extract:
//   - Row index: The last component of the OID (instance identifier)
//   - Column index: The second-to-last component (column identifier)
//
// Parameters:
//   - job: The collection job for storing results and errors
//   - poll: The poll configuration containing the table base OID
func (this *SNMPv2Collector) table(job *l8tpollaris.CJob, poll *l8tpollaris.L8Poll) {
	m := this.walk(job, poll, false)
	if job.Error != "" {
		return
	}
	tbl := &l8tpollaris.CTable{Rows: make(map[int32]*l8tpollaris.CRow), Columns: make(map[int32]string)}
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

// Online returns the connection status of the SNMP collector.
// Returns true only if the session is connected AND at least one poll
// has succeeded. This provides accurate device reachability status.
func (this *SNMPv2Collector) Online() bool {
	return this.connected && this.pollSuccess
}

// getRowAndColName extracts the row index and column identifier from an
// SNMP table OID. For example, from ".1.3.6.1.2.1.2.2.1.6.1", it extracts:
//   - Row index: 1 (the last component, representing the interface index)
//   - Column name: "6" (second-to-last component, representing ifPhysAddress)
//
// Parameters:
//   - oid: The full OID string from an SNMP walk result
//
// Returns:
//   - row: The row index as int32 (-1 if parsing fails)
//   - col: The column identifier as string (empty if parsing fails)
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
