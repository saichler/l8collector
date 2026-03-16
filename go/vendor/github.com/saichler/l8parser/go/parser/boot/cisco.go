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

package boot

import (
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
)

// CreateCiscoSwitchBootPolls creates collection and parsing Pollaris model for Cisco switches
func CreateCiscoSwitchBootPolls() *l8tpollaris.L8Pollaris {
	polaris := &l8tpollaris.L8Pollaris{}
	polaris.Name = "cisco-switch"
	polaris.Groups = []string{"cisco", "cisco-switch"}
	polaris.Polling = make(map[string]*l8tpollaris.L8Poll)
	createCiscoSystemPoll(polaris)
	createCiscoVersionPoll(polaris)
	createCiscoSerialPoll(polaris)
	createCiscoFirmwarePoll(polaris)
	createCiscoInterfacesPoll(polaris)
	createCiscoCpuPoll(polaris)
	createCiscoMemoryPoll(polaris)
	createCiscoTemperaturePoll(polaris)
	createOspfPoll(polaris, "ciscoSwitchOspf")
	createBgpPoll(polaris, "ciscoSwitchBgp")
	createVrfSshPoll(polaris, "ciscoSwitchVrf", "show ip vrf detail", "ios")
	return polaris
}

// CreateCiscoRouterBootPolls creates collection and parsing Pollaris model for Cisco routers
func CreateCiscoRouterBootPolls() *l8tpollaris.L8Pollaris {
	polaris := &l8tpollaris.L8Pollaris{}
	polaris.Name = "cisco-router"
	polaris.Groups = []string{"cisco", "cisco-router"}
	polaris.Polling = make(map[string]*l8tpollaris.L8Poll)
	createCiscoSystemPoll(polaris)
	createCiscoVersionPoll(polaris)
	createCiscoSerialPoll(polaris)
	createCiscoFirmwarePoll(polaris)
	createCiscoInterfacesPoll(polaris)
	createCiscoCpuPoll(polaris)
	createCiscoMemoryPoll(polaris)
	createCiscoTemperaturePoll(polaris)
	createCiscoRoutingPoll(polaris)
	createOspfPoll(polaris, "ciscoRouterOspf")
	createBgpPoll(polaris, "ciscoRouterBgp")
	createVrfSshPoll(polaris, "ciscoRouterVrf", "show vrf all detail", "iosxr")
	return polaris
}

// Cisco device-specific polling functions
func createCiscoSystemPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("ciscoSystem")
	poll.What = ".1.3.6.1.2.1.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	// Equipment Info Attributes
	poll.Attributes = append(poll.Attributes, createCiscoVendor())
	poll.Attributes = append(poll.Attributes, createSysName())
	poll.Attributes = append(poll.Attributes, createSysOid())
	poll.Attributes = append(poll.Attributes, createSysNameAttribute())
	poll.Attributes = append(poll.Attributes, createSysOidAttribute())
	poll.Attributes = append(poll.Attributes, createLocationAttribute())
	poll.Attributes = append(poll.Attributes, createUptimeAttribute())
	poll.Attributes = append(poll.Attributes, createHardwareAttribute())
	poll.Attributes = append(poll.Attributes, createSoftwareAttribute())
	poll.Attributes = append(poll.Attributes, createSeriesAttribute())
	poll.Attributes = append(poll.Attributes, createFamilyAttribute())
	p.Polling[poll.Name] = poll
}

func createCiscoVersionPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("ciscoVersion")
	poll.What = ".1.3.6.1.4.1.9.9.25.1.1.1.2.2"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createCiscoVersion())
	p.Polling[poll.Name] = poll
}

func createCiscoSerialPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("ciscoSerial")
	poll.What = ".1.3.6.1.4.1.9.3.6.3.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createCiscoSerial())
	p.Polling[poll.Name] = poll
}

func createCiscoInterfacesPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("ciscoInterfaces")
	poll.What = ".1.3.6.1.2.1.2.2.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	// Original Interface Attributes
	poll.Attributes = append(poll.Attributes, createInterfaceName())
	poll.Attributes = append(poll.Attributes, createInterfaceStatus())
	poll.Attributes = append(poll.Attributes, createInterfaceSpeed())
	poll.Attributes = append(poll.Attributes, createInterfaceMtu())

	// Enhanced Physical Interface Attributes (nested in ports)
	// Only include attributes whose OIDs are under the walked subtree .1.3.6.1.2.1.2.2.1
	poll.Attributes = append(poll.Attributes, createInterfaceIdAttribute())
	poll.Attributes = append(poll.Attributes, createInterfaceNameAttribute())
	poll.Attributes = append(poll.Attributes, createInterfaceStatusAttribute())
	// ifAlias (.1.3.6.1.2.1.31.1.1.1.18) is in ifXTable, not ifEntry - needs separate poll
	poll.Attributes = append(poll.Attributes, createInterfaceTypeAttribute())
	poll.Attributes = append(poll.Attributes, createInterfaceSpeedAttribute())
	poll.Attributes = append(poll.Attributes, createInterfaceMacAddressAttribute())
	// ipAdEntAddr (.1.3.6.1.2.1.4.20.1.1) is in IP-MIB, not ifEntry - needs separate poll
	poll.Attributes = append(poll.Attributes, createInterfaceMtuAttribute())
	poll.Attributes = append(poll.Attributes, createInterfaceAdminStatusAttribute())

	// Interface Statistics (all under .1.3.6.1.2.1.2.2.1.*)
	poll.Attributes = append(poll.Attributes, createInterfaceRxPacketsAttribute())
	poll.Attributes = append(poll.Attributes, createInterfaceTxPacketsAttribute())
	poll.Attributes = append(poll.Attributes, createInterfaceRxBytesAttribute())
	poll.Attributes = append(poll.Attributes, createInterfaceTxBytesAttribute())
	poll.Attributes = append(poll.Attributes, createInterfaceRxErrorsAttribute())
	poll.Attributes = append(poll.Attributes, createInterfaceTxErrorsAttribute())
	poll.Attributes = append(poll.Attributes, createInterfaceRxDropsAttribute())
	poll.Attributes = append(poll.Attributes, createInterfaceTxDropsAttribute())
	// dot3StatsLateCollisions (.1.3.6.1.2.1.10.7) is in EtherLike-MIB, not ifEntry

	// Logical Interfaces (for VLANs, Loopbacks, etc.)
	poll.Attributes = append(poll.Attributes, createLogicalInterfaceIdAttribute())
	poll.Attributes = append(poll.Attributes, createLogicalInterfaceNameAttribute())
	poll.Attributes = append(poll.Attributes, createLogicalInterfaceStatusAttribute())
	// ifAlias (.1.3.6.1.2.1.31.1.1.1.18) is in ifXTable - needs separate poll
	poll.Attributes = append(poll.Attributes, createLogicalInterfaceTypeAttribute())
	// ipAdEntAddr (.1.3.6.1.2.1.4.20.1.1) is in IP-MIB - needs separate poll
	poll.Attributes = append(poll.Attributes, createLogicalInterfaceMtuAttribute())
	poll.Attributes = append(poll.Attributes, createLogicalInterfaceAdminStatusAttribute())

	p.Polling[poll.Name] = poll
}

func createCiscoCpuPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("ciscoCpu")
	poll.What = ".1.3.6.1.4.1.9.9.109.1.1.1.1.5.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createCpuUtilization())
	p.Polling[poll.Name] = poll
}

func createCiscoMemoryPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("ciscoMemory")
	poll.What = ".1.3.6.1.4.1.9.9.48.1.1.1.6.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createMemoryUtilization())
	p.Polling[poll.Name] = poll
}

func createCiscoRoutingPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("ciscoRouting")
	poll.What = ".1.3.6.1.2.1.4.21.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createRoutingTableEntry())
	p.Polling[poll.Name] = poll
}

// Cisco-specific attribute creation functions
func createCiscoVendor() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.vendor"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createContainsRule("cisco", ".1.3.6.1.2.1.1.1.0", "Cisco"))
	return attr
}

func createCiscoVersion() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.version"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.9.9.25.1.1.1.2.2"))
	return attr
}

func createCiscoTemperaturePoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("ciscoTemperature")
	poll.What = ".1.3.6.1.4.1.9.9.13.1.3.1.3.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createCiscoTemperature())
	p.Polling[poll.Name] = poll
}

func createCiscoTemperature() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.temperature"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.9.9.13.1.3.1.3.1"))
	return attr
}

func createCiscoSerial() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.serialnumber"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.9.3.6.3.0"))
	return attr
}

func createCiscoFirmwarePoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("ciscoFirmware")
	poll.What = ".1.3.6.1.4.1.9.9.25.1.1.1.2.2"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createCiscoFirmwareVersion())
	p.Polling[poll.Name] = poll
}

func createCiscoFirmwareVersion() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.firmwareversion"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.9.9.25.1.1.1.2.2")) // CISCO-IMAGE-MIB
	return attr
}

