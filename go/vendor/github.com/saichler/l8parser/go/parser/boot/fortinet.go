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

// CreateFortinetFirewallBootPolls creates collection and parsing Pollaris model for Fortinet firewalls
func CreateFortinetFirewallBootPolls() *l8tpollaris.L8Pollaris {
	polaris := &l8tpollaris.L8Pollaris{}
	polaris.Name = "fortinet-firewall"
	polaris.Groups = []string{"fortinet", "fortinet-firewall"}
	polaris.Polling = make(map[string]*l8tpollaris.L8Poll)
	createFortinetSystemPoll(polaris)
	createFortinetMibSystemPoll(polaris)
	createFortinetSerialPoll(polaris)
	createFortinetFirmwarePoll(polaris)
	createFortinetInterfacesPoll(polaris)
	createFortinetSessionsPoll(polaris)
	createFortinetVpnPoll(polaris)
	createFortinetCpuPoll(polaris)
	createFortinetMemoryPoll(polaris)
	createFortinetTemperaturePoll(polaris)
	return polaris
}

// Fortinet device-specific polling functions
func createFortinetSystemPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("fortinetSystem")
	poll.What = ".1.3.6.1.4.1.12356.1.1.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createFortinetVersion())
	p.Polling[poll.Name] = poll
}

func createFortinetMibSystemPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("fortinetMibSystem")
	poll.What = ".1.3.6.1.2.1.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createFortinetVendor())
	poll.Attributes = append(poll.Attributes, createSysName())
	p.Polling[poll.Name] = poll
}

func createFortinetInterfacesPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("fortinetInterfaces")
	poll.What = ".1.3.6.1.2.1.2.2.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createInterfaceName())
	poll.Attributes = append(poll.Attributes, createInterfaceStatus())
	p.Polling[poll.Name] = poll
}

func createFortinetSessionsPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("fortinetSessions")
	poll.What = ".1.3.6.1.4.1.12356.101.4.1.8.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createFortinetActiveSessions())
	p.Polling[poll.Name] = poll
}

func createFortinetVpnPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("fortinetVpn")
	poll.What = ".1.3.6.1.4.1.12356.101.12.2.3.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createFortinetVpnTunnelStatus())
	p.Polling[poll.Name] = poll
}

func createFortinetCpuPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("fortinetCpu")
	poll.What = ".1.3.6.1.4.1.12356.101.4.1.3.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createFortinetCpuUtilization())
	p.Polling[poll.Name] = poll
}

func createFortinetCpuUtilization() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.performance.cpuusagepercent"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.12356.101.4.1.3.0"))
	return attr
}

func createFortinetMemoryPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("fortinetMemory")
	poll.What = ".1.3.6.1.4.1.12356.101.4.1.4.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createFortinetMemoryUtilization())
	p.Polling[poll.Name] = poll
}

func createFortinetMemoryUtilization() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.performance.memoryusagepercent"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.12356.101.4.1.4.0"))
	return attr
}

func createFortinetTemperaturePoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("fortinetTemperature")
	poll.What = ".1.3.6.1.4.1.12356.101.4.3.1.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createFortinetTemperature())
	p.Polling[poll.Name] = poll
}

func createFortinetTemperature() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.temperature"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.12356.101.4.3.1.0"))
	return attr
}

// Fortinet-specific attribute creation functions
func createFortinetVendor() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.vendor"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createContainsRule("fortinet", ".1.3.6.1.2.1.1.1.0", "Fortinet"))
	return attr
}

func createFortinetVersion() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.version"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.12356.1.1.0"))
	return attr
}

func createFortinetActiveSessions() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.performance.activeconnections"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.12356.101.4.1.8.0"))
	return attr
}

func createFortinetSerialPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("fortinetSerial")
	poll.What = ".1.3.6.1.4.1.12356.100.1.2.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createFortinetSerial())
	p.Polling[poll.Name] = poll
}

func createFortinetSerial() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.serialnumber"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.12356.100.1.2.0")) // fgSysSerial
	return attr
}

func createFortinetFirmwarePoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("fortinetFirmware")
	poll.What = ".1.3.6.1.4.1.12356.100.1.1.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createFortinetFirmwareVersion())
	p.Polling[poll.Name] = poll
}

func createFortinetFirmwareVersion() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.firmwareversion"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.12356.100.1.1.0")) // fgSysVersion
	return attr
}

func createFortinetVpnTunnelStatus() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.networklinks.linkstatus"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.12356.101.12.2.3.1.3.1"))
	return attr
}
