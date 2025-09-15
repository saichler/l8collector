// +build !cgo

package snmp

import "fmt"

// SNMPSession represents a fallback SNMP session when CGO is disabled
type SNMPSession struct {
	host      string
	community string
}

// NewSNMPSession creates a fallback SNMP session that returns an error
func NewSNMPSession(host, community string) (*SNMPSession, error) {
	return nil, fmt.Errorf("net-snmp CGO bindings not available - rebuild with CGO_ENABLED=1 and libsnmp-dev installed")
}

// Walk returns an error since CGO is disabled
func (s *SNMPSession) Walk(oid string) ([]SnmpPDU, error) {
	return nil, fmt.Errorf("net-snmp CGO bindings not available")
}

// Close is a no-op for the fallback implementation
func (s *SNMPSession) Close() error {
	return nil
}