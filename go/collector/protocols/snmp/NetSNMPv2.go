package snmp

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/saichler/l8pollaris/go/types/l8poll"
	"github.com/saichler/l8types/go/ifs"
)

type NetSNMPCollector struct {
	config    *l8poll.L8T_Connection
	resources ifs.IResources
}

func NewNetSNMPCollector(config *l8poll.L8T_Connection, resources ifs.IResources) *NetSNMPCollector {
	return &NetSNMPCollector{
		config:    config,
		resources: resources,
	}
}

func (n *NetSNMPCollector) snmpWalk(oid string) ([]SnmpPDU, error) {
	if n.config == nil {
		return nil, fmt.Errorf("SNMP config is not initialized")
	}

	timeout := n.config.Timeout
	if timeout == 0 {
		timeout = 60 // Default 60 seconds
	}

	args := []string{
		"-v", "2c",
		"-c", n.config.ReadCommunity,
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