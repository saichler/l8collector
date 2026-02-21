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
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8types/go/ifs"
)

// NetSNMPCollector provides a fallback SNMP implementation using the net-snmp
// command-line tools (snmpwalk). This is used when the WapSNMP library times out
// or fails, providing enhanced compatibility with certain devices.
//
// The net-snmp tools are widely deployed and have excellent device compatibility,
// making them a reliable fallback for problematic SNMP implementations.
type NetSNMPCollector struct {
	config    *l8tpollaris.L8PHostProtocol  // Host configuration with address and credentials
	resources ifs.IResources                // Layer8 resources for logging and security
}

// NewNetSNMPCollector creates a new NetSNMPCollector instance with the provided
// configuration and resources. This is typically called as a fallback when
// the primary WapSNMP-based collector times out.
//
// Parameters:
//   - config: Host protocol configuration containing address, port, and credential ID
//   - resources: Layer8 resources for accessing security credentials and logging
//
// Returns:
//   - A new NetSNMPCollector instance ready for use
func NewNetSNMPCollector(config *l8tpollaris.L8PHostProtocol, resources ifs.IResources) *NetSNMPCollector {
	return &NetSNMPCollector{
		config:    config,
		resources: resources,
	}
}

// snmpWalk performs an SNMP walk using the net-snmp snmpwalk command-line tool.
// It retrieves the community string from the security service and executes
// snmpwalk with appropriate flags for SNMP v2c.
//
// The command is executed with:
//   - SNMP version 2c
//   - Configured timeout with 3 retries
//   - Numeric OID output format (-On)
//   - Quick print format (-Oq)
//
// Parameters:
//   - oid: The base OID to walk from
//
// Returns:
//   - Slice of SnmpPDU containing all OID-value pairs found
//   - error if config is nil, command fails, or parsing fails
func (n *NetSNMPCollector) snmpWalk(oid string) ([]SnmpPDU, error) {
	if n.config == nil {
		return nil, fmt.Errorf("SNMP config is not initialized")
	}

	timeout := n.config.Timeout
	if timeout == 0 {
		timeout = 60 // Default 60 seconds
	}

	if n.resources != nil && n.resources.Logger() != nil {
		n.resources.Logger().Debug("net-snmp timeout configured to: ", timeout, " seconds")
	}

	_, readCommunity, _, _, e := n.resources.Security().Credential(n.config.CredId, "snmp", n.resources)
	if e != nil {
		panic(e)
	}

	args := []string{
		"-v", "2c",
		"-c", readCommunity,
		"-t", strconv.Itoa(int(timeout)),
		"-r", "3", // 3 retries
		"-On", // Numeric OIDs
		"-Oq", // Quick print
		n.config.Addr + ":" + strconv.Itoa(int(n.config.Port)),
		oid,
	}

	if n.resources != nil && n.resources.Logger() != nil {
		n.resources.Logger().Debug("Executing net-snmp snmpwalk with args: ", strings.Join(args, " "))
	}

	cmd := exec.Command("snmpwalk", args...)

	// Set a timeout for the command execution
	cmdTimeout := time.Duration(timeout+5) * time.Second
	done := make(chan error, 1)
	var output []byte
	var err error

	go func() {
		output, err = cmd.CombinedOutput()
		done <- err
	}()

	select {
	case cmdErr := <-done:
		if cmdErr != nil {
			return nil, fmt.Errorf("net-snmp snmpwalk failed: %v, output: %s", cmdErr, string(output))
		}
	case <-time.After(cmdTimeout):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return nil, fmt.Errorf("net-snmp snmpwalk timed out after %s", cmdTimeout.String())
	}

	if len(output) == 0 {
		return nil, fmt.Errorf("net-snmp snmpwalk returned no data for OID %s", oid)
	}

	return n.parseSnmpWalkOutput(string(output))
}

// snmpGet performs a single SNMP GET using the net-snmp snmpget command-line tool.
// It retrieves the value for a specific OID rather than walking an entire subtree.
//
// Parameters:
//   - oid: The OID to get
//
// Returns:
//   - SnmpPDU containing the OID and its value
//   - error if config is nil, command fails, or parsing fails
func (n *NetSNMPCollector) snmpGet(oid string) (*SnmpPDU, error) {
	if n.config == nil {
		return nil, fmt.Errorf("SNMP config is not initialized")
	}

	timeout := n.config.Timeout
	if timeout == 0 {
		timeout = 60
	}

	_, readCommunity, _, _, e := n.resources.Security().Credential(n.config.CredId, "snmp", n.resources)
	if e != nil {
		panic(e)
	}

	args := []string{
		"-v", "2c",
		"-c", readCommunity,
		"-t", strconv.Itoa(int(timeout)),
		"-r", "3",
		"-On",
		"-Oq",
		n.config.Addr + ":" + strconv.Itoa(int(n.config.Port)),
		oid,
	}

	cmd := exec.Command("snmpget", args...)

	cmdTimeout := time.Duration(timeout+5) * time.Second
	done := make(chan error, 1)
	var output []byte
	var err error

	go func() {
		output, err = cmd.CombinedOutput()
		done <- err
	}()

	select {
	case cmdErr := <-done:
		if cmdErr != nil {
			return nil, fmt.Errorf("net-snmp snmpget failed: %v, output: %s", cmdErr, string(output))
		}
	case <-time.After(cmdTimeout):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return nil, fmt.Errorf("net-snmp snmpget timed out after %s", cmdTimeout.String())
	}

	if len(output) == 0 {
		return nil, fmt.Errorf("net-snmp snmpget returned no data for OID %s", oid)
	}

	pdus, parseErr := n.parseSnmpWalkOutput(string(output))
	if parseErr != nil || len(pdus) == 0 {
		return nil, fmt.Errorf("failed to parse snmpget output for OID %s", oid)
	}

	return &pdus[0], nil
}

// parseSnmpWalkOutput parses the text output from the snmpwalk command
// into a slice of SnmpPDU structures. Each line is expected to be in the
// format "OID VALUE" as produced by snmpwalk with -Oq flag.
//
// Parameters:
//   - output: The raw text output from snmpwalk command
//
// Returns:
//   - Slice of SnmpPDU with parsed OID-value pairs
//   - error if no valid data could be parsed
func (n *NetSNMPCollector) parseSnmpWalkOutput(output string) ([]SnmpPDU, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var pdus []SnmpPDU

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}

		oidStr := strings.TrimSpace(parts[0])
		valueStr := strings.TrimSpace(parts[1])

		// Parse the value based on its type
		value := n.parseValue(valueStr)

		pdus = append(pdus, SnmpPDU{
			Name:  oidStr,
			Value: value,
		})
	}

	if len(pdus) == 0 {
		return nil, fmt.Errorf("failed to parse any valid SNMP data from output")
	}

	return pdus, nil
}

// parseValue converts a string value from snmpwalk output to its appropriate
// Go type. It recognizes standard net-snmp type indicators and converts them:
//   - STRING: -> string
//   - INTEGER: -> int64
//   - Counter32: -> uint64
//   - Counter64: -> uint64
//   - Gauge32: -> uint64
//   - TimeTicks: -> uint64 (extracted from parentheses)
//   - OID: -> string
//   - IpAddress: -> string
//   - Hex-STRING: -> string
//
// Values without type indicators are attempted as integers, falling back to strings.
//
// Parameters:
//   - valueStr: The value string from snmpwalk output, potentially with type prefix
//
// Returns:
//   - The parsed value in the appropriate Go type
func (n *NetSNMPCollector) parseValue(valueStr string) interface{} {
	// Remove common net-snmp type indicators
	if strings.Contains(valueStr, "STRING: ") {
		return strings.TrimPrefix(valueStr, "STRING: ")
	}
	if strings.Contains(valueStr, "INTEGER: ") {
		intStr := strings.TrimPrefix(valueStr, "INTEGER: ")
		if val, err := strconv.ParseInt(intStr, 10, 64); err == nil {
			return val
		}
		return intStr
	}
	if strings.Contains(valueStr, "Counter32: ") {
		counterStr := strings.TrimPrefix(valueStr, "Counter32: ")
		if val, err := strconv.ParseUint(counterStr, 10, 32); err == nil {
			return val
		}
		return counterStr
	}
	if strings.Contains(valueStr, "Counter64: ") {
		counterStr := strings.TrimPrefix(valueStr, "Counter64: ")
		if val, err := strconv.ParseUint(counterStr, 10, 64); err == nil {
			return val
		}
		return counterStr
	}
	if strings.Contains(valueStr, "Gauge32: ") {
		gaugeStr := strings.TrimPrefix(valueStr, "Gauge32: ")
		if val, err := strconv.ParseUint(gaugeStr, 10, 32); err == nil {
			return val
		}
		return gaugeStr
	}
	if strings.Contains(valueStr, "TimeTicks: ") {
		ticksStr := strings.TrimPrefix(valueStr, "TimeTicks: ")
		// TimeTicks often has format like "TimeTicks: (12345) 0:02:03.45"
		if idx := strings.Index(ticksStr, ")"); idx != -1 {
			ticksStr = strings.TrimSpace(ticksStr[:idx])
			ticksStr = strings.TrimPrefix(ticksStr, "(")
			if val, err := strconv.ParseUint(ticksStr, 10, 32); err == nil {
				return val
			}
		}
		return ticksStr
	}
	if strings.Contains(valueStr, "OID: ") {
		return strings.TrimPrefix(valueStr, "OID: ")
	}
	if strings.Contains(valueStr, "IpAddress: ") {
		return strings.TrimPrefix(valueStr, "IpAddress: ")
	}
	if strings.Contains(valueStr, "Hex-STRING: ") {
		return strings.TrimPrefix(valueStr, "Hex-STRING: ")
	}

	// If no type indicator found, try to parse as integer, otherwise return as string
	if val, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
		return val
	}

	return valueStr
}
