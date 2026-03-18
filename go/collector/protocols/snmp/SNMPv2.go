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
	"strings"
	"time"

	wapsnmp "github.com/cdevr/WapSNMP"
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

	session, err := wapsnmp.NewWapSNMP(target, community, version, timeout, 1)
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

	for attempt := 1; attempt <= 10; attempt++ {
		pdu = nil
		lastError = nil

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		done := make(chan bool, 1)

		go func() {
			defer func() {
				if r := recover(); r != nil {
					lastError = fmt.Errorf("SNMP GET panic (session closed during operation): %v", r)
				}
				done <- true
			}()
			p, e := this.snmpGet(poll.What)
			if e == nil {
				pdu = p
			} else {
				lastError = e
			}
		}()

		select {
		case <-done:
			this.pollSuccess = true
			cancel()
		case <-ctx.Done():
			cancel()
			// Close session to stop the abandoned goroutine, then reconnect on next Exec.
			if this.session != nil {
				this.session.Close()
				this.session = nil
			}
			this.connected = false
			lastError = fmt.Errorf("timeout after %s", timeout.String())
		}

		// Success — no need to retry
		if lastError == nil {
			break
		}

		// First attempt failed — sleep 1s, reconnect, and retry
		if attempt < 10 {
			if this.resources != nil && this.resources.Logger() != nil {
				this.resources.Logger().Warning("SNMP GET failed for ", this.config.Addr,
					" OID: ", poll.What, " error: ", lastError.Error(), ". Sleeping 1s and retrying.")
			}
			time.Sleep(1 * time.Second)
			if reconnErr := this.reconnectSession(); reconnErr != nil {
				break // Can't reconnect, no point retrying
			}
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
// On failure, it closes the session, waits 1 second, reconnects, and retries once.
//
// Parameters:
//   - oid: The OID to get (e.g., ".1.3.6.1.2.1.1.1.0")
//
// Returns:
//   - SnmpPDU containing the OID and its value
//   - error if session is not initialized or GET fails after retry
func (this *SNMPv2Collector) snmpGet(oid string) (*SnmpPDU, error) {
	if this.session == nil {
		return nil, fmt.Errorf("SNMP session is not initialized")
	}

	parsedOid, err := wapsnmp.ParseOid(oid)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OID %s: %v", oid, err)
	}

	value, err := this.session.Get(parsedOid)
	if err == nil {
		return &SnmpPDU{Name: oid, Value: value}, nil
	}

	// If the session was closed externally (by the timeout handler),
	// do NOT reconnect — that would leak a UDP socket. Just bail out.
	if this.session == nil {
		return nil, fmt.Errorf("SNMP session was closed during get for OID %s", oid)
	}

	// First attempt failed — reconnect and retry once
	if this.resources != nil && this.resources.Logger() != nil {
		this.resources.Logger().Warning("SNMP GET failed for ", this.config.Addr,
			" OID ", oid, ": ", err.Error(), ". Reconnecting and retrying.")
	}
	if reconnErr := this.reconnectSession(); reconnErr != nil {
		return nil, fmt.Errorf("SNMP GET failed for OID %s: %v (reconnect also failed: %v)", oid, err, reconnErr)
	}
	if this.session == nil {
		return nil, fmt.Errorf("SNMP session is nil after reconnect for OID %s", oid)
	}

	value, err = this.session.Get(parsedOid)
	if err != nil {
		return nil, fmt.Errorf("SNMP GET failed for OID %s after reconnect: %v", oid, err)
	}

	return &SnmpPDU{Name: oid, Value: value}, nil
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

	for attempt := 1; attempt <= 10; attempt++ {
		pdus = nil
		lastError = nil

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		var e error
		done := make(chan bool, 1)

		go func() {
			defer func() {
				if r := recover(); r != nil {
					e = fmt.Errorf("SNMP Walk panic (session closed during operation): %v", r)
				}
				done <- true
			}()
			pdus, e = this.snmpWalk(poll.What)
		}()

		select {
		case <-done:
			this.pollSuccess = true
			cancel()
			if e != nil {
				lastError = e
			}

		case <-ctx.Done():
			cancel()
			// Timeout occurred - close the session to stop the abandoned goroutine,
			// then reconnect so the next job gets a fresh connection.
			if this.session != nil {
				this.session.Close()
				this.session = nil
			}
			this.connected = false
			lastError = fmt.Errorf("timeout after %s", timeout.String())
		}

		// Success — no need to retry
		if lastError == nil {
			break
		}

		// First attempt failed — sleep 1s, reconnect, and retry
		if attempt < 10 {
			if this.resources != nil && this.resources.Logger() != nil {
				this.resources.Logger().Warning("SNMP Walk failed for ", this.config.Addr,
					" OID: ", poll.What, " error: ", lastError.Error(), ". Sleeping 1s and retrying.")
			}
			time.Sleep(1 * time.Second)
			if reconnErr := this.reconnectSession(); reconnErr != nil {
				break // Can't reconnect, no point retrying
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

