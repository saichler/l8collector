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

package snmp

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	wapsnmp "github.com/cdevr/WapSNMP"
	"github.com/saichler/l8collector/go/collector/protocols"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8srlz/go/serialize/object"
)

// snmpWalk performs the actual SNMP walk using WapSNMP's GetNext operations.
// It iteratively retrieves OIDs within the specified subtree until it reaches
// an OID outside the subtree or encounters an error. On a GetNext failure,
// it reconnects the session and retries from the last successful OID.
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
		// Session may have been closed by the timeout handler in walk()
		if this.session == nil {
			return pdus, fmt.Errorf("SNMP session was closed during walk")
		}
		nextOid, value, err := this.session.GetNext(currentOid)
		if err != nil {
			// Try reconnecting and retrying this one GetNext
			if this.resources != nil && this.resources.Logger() != nil {
				this.resources.Logger().Warning("SNMP GetNext failed for ", this.config.Addr,
					" OID ", currentOid.String(), ": ", err.Error(), ". Reconnecting and retrying.")
			}
			if reconnErr := this.reconnectSession(); reconnErr != nil {
				break // Can't recover, stop walk with what we have
			}
			if this.session == nil {
				return pdus, fmt.Errorf("SNMP session was closed during walk")
			}
			nextOid, value, err = this.session.GetNext(currentOid)
			if err != nil {
				break // Still failing after reconnect, stop walk
			}
		}

		// Check if we're still within the requested subtree
		if !nextOid.Within(parsedOid) {
			break // We've walked beyond the requested subtree
		}

		// BERType values indicate WapSNMP misinterpreted the response
		// (e.g., endOfMibView). Stop the walk.
		if _, isBER := value.(wapsnmp.BERType); isBER {
			break
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

// reconnectSession closes the current SNMP session, waits 1 second for the
// socket to fully release, and opens a fresh connection to the same target.
func (this *SNMPv2Collector) reconnectSession() error {
	this.Disconnect()
	time.Sleep(1 * time.Second)
	return this.Connect()
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
