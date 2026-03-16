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

package rules

import (
	"fmt"
	"strings"

	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8types/go/ifs"
	types2 "github.com/saichler/probler/go/types"
)

// InferDeviceType is a parsing rule that automatically determines the device type
// (router, switch, firewall, server, etc.) based on sysOID, sysDescr, and vendor information.
// It analyzes SNMP system information to classify the network device.
type InferDeviceType struct{}

// Name returns the rule identifier "InferDeviceType".
func (this *InferDeviceType) Name() string {
	return "InferDeviceType"
}

// ParamNames returns the required parameter names for this rule.
func (this *InferDeviceType) ParamNames() []string {
	return []string{}
}

// DeviceType enum values from proto
const (
	DEVICE_TYPE_UNKNOWN       = 0
	DEVICE_TYPE_ROUTER        = 1
	DEVICE_TYPE_SWITCH        = 2
	DEVICE_TYPE_FIREWALL      = 3
	DEVICE_TYPE_LOAD_BALANCER = 4
	DEVICE_TYPE_ACCESS_POINT  = 5
	DEVICE_TYPE_SERVER        = 6
	DEVICE_TYPE_STORAGE       = 7
	DEVICE_TYPE_GATEWAY       = 8
)

func (this *InferDeviceType) Parse(resources ifs.IResources, workSpace map[string]interface{}, params map[string]*l8tpollaris.L8PParameter, any interface{}, pollWhat string) error {
	// Get the NetworkDevice
	networkDevice, ok := any.(*types2.NetworkDevice)
	if !ok {
		return resources.Logger().Error("Target object is not a NetworkDevice")
	}

	// Ensure Equipmentinfo exists (note: lowercase 'i')
	if networkDevice.Equipmentinfo == nil {
		networkDevice.Equipmentinfo = &types2.EquipmentInfo{}
	}

	// Get sysObjectID from input using GetValueInput (like other parsing rules)
	input := workSpace[Input]
	if input == nil {
		return resources.Logger().Error("nil input for InferDeviceType")
	}

	// Get the sysObjectID value using the same pattern as Set rule
	value, _, err := GetValueInput(resources, input, params, pollWhat)
	if err != nil {
		return resources.Logger().Error("Error getting sysObjectID:", err)
	}

	if value == nil {
		return resources.Logger().Error("nil sysObjectID value")
	}

	sysObjectID := convertInterfaceToString(value)

	// Infer device type based on sysObjectID patterns
	deviceType := inferDeviceTypeFromOID(sysObjectID)
	networkDevice.Equipmentinfo.DeviceType = types2.DeviceType(deviceType)

	return nil
}

func inferDeviceTypeFromOID(sysObjectID string) int32 {
	sysObjectIDLower := strings.ToLower(sysObjectID)

	// OID-based detection (vendor-specific OIDs)
	// Cisco (1.3.6.1.4.1.9) - Most common enterprise network vendor
	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.9") {
		// Cisco routers typically have specific sub-OIDs
		if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.9.1.1") || // Old Cisco routers
			strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.9.12") || // ISR/ASR series
			strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.9.1.222") { // Newer routers
			return DEVICE_TYPE_ROUTER
		}
		// Cisco switches
		if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.9.1.23") || // Catalyst switches
			strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.9.1.516") || // Catalyst 2960
			strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.9.1.717") || // Catalyst 3750
			strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.9.1.1208") || // Catalyst 4500 series
			strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.9.1.1404") { // Additional Catalyst series
			return DEVICE_TYPE_SWITCH
		}
		// Cisco ASA firewalls
		if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.9.1.745") || // ASA 5500 series
			strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.9.1.1069") { // ASA 5585
			return DEVICE_TYPE_FIREWALL
		}
		// Default Cisco to router (most common)
		return DEVICE_TYPE_ROUTER
	}

	// Juniper (1.3.6.1.4.1.2636) - Primarily routing/switching
	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.2636") {
		// Juniper is primarily routing equipment
		return DEVICE_TYPE_ROUTER
	}

	// Palo Alto (1.3.6.1.4.1.25461) - Firewalls
	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.25461") {
		return DEVICE_TYPE_FIREWALL
	}

	// Fortinet (1.3.6.1.4.1.12356) - Firewalls
	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.12356") {
		return DEVICE_TYPE_FIREWALL
	}

	// F5 (1.3.6.1.4.1.3375) - Load Balancers
	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.3375") {
		return DEVICE_TYPE_LOAD_BALANCER
	}

	// Arista (1.3.6.1.4.1.30065) - Switches
	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.30065") {
		return DEVICE_TYPE_SWITCH
	}

	// Dell servers (1.3.6.1.4.1.674)
	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.674") {
		return DEVICE_TYPE_SERVER
	}

	// HP/HPE (1.3.6.1.4.1.232)
	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.232") {
		// HP makes both servers and network equipment
		// Default to server for HP as that's more common for SNMP management
		return DEVICE_TYPE_SERVER
	}

	// Ubiquiti (1.3.6.1.4.1.41112) - Access Points
	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.41112") {
		return DEVICE_TYPE_ACCESS_POINT
	}

	// NetApp (1.3.6.1.4.1.789) - Storage
	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.789") {
		return DEVICE_TYPE_STORAGE
	}

	// EMC (1.3.6.1.4.1.1139) - Storage
	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.1139") {
		return DEVICE_TYPE_STORAGE
	}

	// Huawei (1.3.6.1.4.1.2011) - Routers and Switches
	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.2011") {
		// Huawei NE series are typically routers
		if strings.Contains(sysObjectIDLower, "ne") {
			return DEVICE_TYPE_ROUTER
		}
		// Default Huawei to router (most common in enterprise)
		return DEVICE_TYPE_ROUTER
	}

	// NEC (1.3.6.1.4.1.119) - Routers
	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.119") {
		return DEVICE_TYPE_ROUTER
	}

	// Check Point (1.3.6.1.4.1.2620) - Firewalls
	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.2620") {
		return DEVICE_TYPE_FIREWALL
	}

	// Nokia/Alcatel-Lucent (1.3.6.1.4.1.6527) - Routers
	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.6527") {
		return DEVICE_TYPE_ROUTER
	}

	// D-Link (1.3.6.1.4.1.171) - Switches and Network Equipment
	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.171") {
		// Most D-Link enterprise OIDs are switches
		// The specific OID 1.3.6.1.4.1.171.10.139.2.1 is typically a managed switch
		return DEVICE_TYPE_SWITCH
	}

	// FlexRadio Systems (1.3.6.1.4.1.8741) - Radio/Access Point equipment
	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.8741") {
		return DEVICE_TYPE_ACCESS_POINT
	}

	// Extreme Networks (1.3.6.1.4.1.1916) - Switches
	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.1916") {
		return DEVICE_TYPE_SWITCH
	}

	// IBM (1.3.6.1.4.1.2) - Servers and Enterprise Systems
	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.2") {
		return DEVICE_TYPE_SERVER
	}

	// Pattern-based detection fallback for device descriptions
	deviceStatus := inferDeviceTypeFromDescriptionPatterns(sysObjectID)
	if deviceStatus != DEVICE_TYPE_UNKNOWN {
		return deviceStatus
	}

	// Default to unknown if no patterns match
	return DEVICE_TYPE_UNKNOWN
}

func inferDeviceTypeFromData(sysDescr string, sysObjectID string) int32 {
	sysDescrLower := strings.ToLower(sysDescr)
	sysObjectIDLower := strings.ToLower(sysObjectID)

	// Router patterns (highest priority for routers)
	routerPatterns := []string{
		"router", "asr", "isr", "7200", "7300", "7400", "7500", "7600",
		"nexus", "mx", "ex", "qfx", "srx", "vmx", "vqfx", "routing",
		"ospf", "bgp", "mpls", "cisco ios", "junos", "nx-os",
		"ne8000", "ix3315", "crs-x", "7750", "7450", "vrp", "ios-xr",
	}

	// Switch patterns
	switchPatterns := []string{
		"switch", "catalyst", "3750", "3560", "2960", "4500", "6500",
		"switching", "vlan", "spanning-tree", "stp", "ethernet switch",
		"7280r3", "arista", "eos", "nexus 9500",
		"d-link", "dgs", "des", "dxs", "dlink",
		"extreme", "exos", "summit", "x440", "x460", "x480",
	}

	// Firewall patterns
	firewallPatterns := []string{
		"firewall", "asa", "pix", "fortigate", "palo alto", "checkpoint",
		"fortios", "pan-os", "security", "utm", "ngfw", "threat",
		"15600", "gaia", "600e",
	}

	// Load balancer patterns
	loadBalancerPatterns := []string{
		"load balancer", "f5", "big-ip", "netscaler", "citrix", "a10",
		"loadbalancer", "application delivery", "adc",
	}

	// Access point patterns
	accessPointPatterns := []string{
		"access point", "wireless", "wifi", "ap", "wap", "aironet",
		"aruba", "ubiquiti", "unifi",
		"flexradio", "flex radio", "radio",
	}

	// Server patterns
	serverPatterns := []string{
		"server", "dell", "hp", "hpe", "supermicro", "lenovo", "ibm",
		"proliant", "poweredge", "system x", "blade", "rack",
		"dl380", "r750", "idrac", "ilo", "gen10",
		"aix", "pseries", "zseries", "mainframe",
	}

	// Storage patterns
	storagePatterns := []string{
		"storage", "san", "nas", "netapp", "emc", "dell emc", "pure storage",
		"hitachi", "hds", "vnx", "unity", "isilon", "compellent",
	}

	// Gateway patterns
	gatewayPatterns := []string{
		"gateway", "edge", "branch", "wan optimizer", "sd-wan", "viptela",
	}

	// OID-based detection (vendor-specific OIDs)
	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.9") { // Cisco
		if containsAnyPattern(sysDescrLower, routerPatterns) {
			return DEVICE_TYPE_ROUTER
		}
		if containsAnyPattern(sysDescrLower, switchPatterns) {
			return DEVICE_TYPE_SWITCH
		}
		if containsAnyPattern(sysDescrLower, firewallPatterns) {
			return DEVICE_TYPE_FIREWALL
		}
	}

	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.2636") { // Juniper
		return DEVICE_TYPE_ROUTER // Juniper is primarily routing/switching
	}

	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.25461") { // Palo Alto
		return DEVICE_TYPE_FIREWALL
	}

	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.12356") { // Fortinet
		return DEVICE_TYPE_FIREWALL
	}

	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.3375") { // F5
		return DEVICE_TYPE_LOAD_BALANCER
	}

	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.674") { // Dell
		return DEVICE_TYPE_SERVER
	}

	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.232") { // HP/HPE
		return DEVICE_TYPE_SERVER
	}

	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.2011") { // Huawei
		return DEVICE_TYPE_ROUTER
	}

	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.119") { // NEC
		return DEVICE_TYPE_ROUTER
	}

	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.2620") { // Check Point
		return DEVICE_TYPE_FIREWALL
	}

	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.6527") { // Nokia/Alcatel-Lucent
		return DEVICE_TYPE_ROUTER
	}

	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.171") { // D-Link
		return DEVICE_TYPE_SWITCH
	}

	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.8741") { // FlexRadio Systems
		return DEVICE_TYPE_ACCESS_POINT
	}

	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.1916") { // Extreme Networks
		return DEVICE_TYPE_SWITCH
	}

	if strings.Contains(sysObjectIDLower, "1.3.6.1.4.1.2") { // IBM
		return DEVICE_TYPE_SERVER
	}

	// Pattern-based detection (fallback to description analysis)
	if containsAnyPattern(sysDescrLower, routerPatterns) {
		return DEVICE_TYPE_ROUTER
	}

	if containsAnyPattern(sysDescrLower, switchPatterns) {
		return DEVICE_TYPE_SWITCH
	}

	if containsAnyPattern(sysDescrLower, firewallPatterns) {
		return DEVICE_TYPE_FIREWALL
	}

	if containsAnyPattern(sysDescrLower, loadBalancerPatterns) {
		return DEVICE_TYPE_LOAD_BALANCER
	}

	if containsAnyPattern(sysDescrLower, accessPointPatterns) {
		return DEVICE_TYPE_ACCESS_POINT
	}

	if containsAnyPattern(sysDescrLower, serverPatterns) {
		return DEVICE_TYPE_SERVER
	}

	if containsAnyPattern(sysDescrLower, storagePatterns) {
		return DEVICE_TYPE_STORAGE
	}

	if containsAnyPattern(sysDescrLower, gatewayPatterns) {
		return DEVICE_TYPE_GATEWAY
	}

	// Default to unknown if no patterns match
	return DEVICE_TYPE_UNKNOWN
}

func containsAnyPattern(text string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}
	return false
}

func inferDeviceTypeFromDescriptionPatterns(sysObjectID string) int32 {
	sysObjectIDLower := strings.ToLower(sysObjectID)

	// Vendor-specific model patterns in OID or description
	routerPatterns := []string{
		"ne8000", "ix3315", "mx240", "mx960", "asr", "isr", "crs",
		"7750", "7450", "router", "junos", "vrp", "ios-xr",
	}

	switchPatterns := []string{
		"7280", "catalyst", "nexus", "ex4300", "qfx", "switch",
		"switching", "7280r3", "eos",
		"d-link", "dgs", "des", "dxs", "dlink",
		"extreme", "exos", "summit", "x440", "x460", "x480",
	}

	firewallPatterns := []string{
		"15600", "checkpoint", "fortigate", "palo alto", "asa",
		"firewall", "fortios", "gaia", "pan-os",
	}

	serverPatterns := []string{
		"poweredge", "proliant", "server", "idrac", "ilo",
		"dl380", "r750", "blade",
		"aix", "pseries", "zseries", "mainframe", "ibm",
	}

	loadBalancerPatterns := []string{
		"big-ip", "f5", "netscaler", "load balancer", "adc",
	}

	accessPointPatterns := []string{
		"wireless", "access point", "wifi", "aironet", "unifi",
		"flexradio", "flex radio", "radio",
	}

	storagePatterns := []string{
		"netapp", "emc", "storage", "san", "nas", "vnx", "unity",
	}

	// Check patterns
	if containsAnyPattern(sysObjectIDLower, routerPatterns) {
		return DEVICE_TYPE_ROUTER
	}
	if containsAnyPattern(sysObjectIDLower, switchPatterns) {
		return DEVICE_TYPE_SWITCH
	}
	if containsAnyPattern(sysObjectIDLower, firewallPatterns) {
		return DEVICE_TYPE_FIREWALL
	}
	if containsAnyPattern(sysObjectIDLower, serverPatterns) {
		return DEVICE_TYPE_SERVER
	}
	if containsAnyPattern(sysObjectIDLower, loadBalancerPatterns) {
		return DEVICE_TYPE_LOAD_BALANCER
	}
	if containsAnyPattern(sysObjectIDLower, accessPointPatterns) {
		return DEVICE_TYPE_ACCESS_POINT
	}
	if containsAnyPattern(sysObjectIDLower, storagePatterns) {
		return DEVICE_TYPE_STORAGE
	}

	return DEVICE_TYPE_UNKNOWN
}

func convertInterfaceToString(value interface{}) string {
	if value == nil {
		return ""
	}

	// Handle byte array to string conversion
	if byteArray, ok := value.([]uint8); ok {
		return strings.TrimSpace(string(byteArray))
	}

	return strings.TrimSpace(fmt.Sprintf("%v", value))
}
