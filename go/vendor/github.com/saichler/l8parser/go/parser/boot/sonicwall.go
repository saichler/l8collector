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

// CreateSonicWallFirewallBootPolls creates collection and parsing Pollaris model for SonicWall firewalls
func CreateSonicWallFirewallBootPolls() *l8tpollaris.L8Pollaris {
	polaris := &l8tpollaris.L8Pollaris{}
	polaris.Name = "sonicwall-firewall"
	polaris.Groups = []string{"sonicwall", "sonicwall-firewall"}
	polaris.Polling = make(map[string]*l8tpollaris.L8Poll)
	createSonicWallSystemPoll(polaris)
	createSonicWallMibSystemPoll(polaris)
	createSonicWallSerialPoll(polaris)
	createSonicWallInterfacesPoll(polaris)
	createSonicWallCpuPoll(polaris)
	createSonicWallMemoryPoll(polaris)
	createSonicWallTemperaturePoll(polaris)
	return polaris
}

// SonicWall device-specific polling functions
func createSonicWallSystemPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("sonicWallSystem")
	poll.What = ".1.3.6.1.4.1.8714.2.1.1.1.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createSonicWallVersion())
	p.Polling[poll.Name] = poll
}

func createSonicWallMibSystemPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("sonicWallMibSystem")
	poll.What = ".1.3.6.1.2.1.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createSonicWallVendor())
	poll.Attributes = append(poll.Attributes, createSysName())
	p.Polling[poll.Name] = poll
}

func createSonicWallInterfacesPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("sonicWallInterfaces")
	poll.What = ".1.3.6.1.2.1.2.2.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createInterfaceName())
	poll.Attributes = append(poll.Attributes, createInterfaceStatus())
	p.Polling[poll.Name] = poll
}

func createSonicWallCpuPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("sonicWallCpu")
	poll.What = ".1.3.6.1.4.1.8714.2.1.3.1.1.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createSonicWallCpuUtilization())
	p.Polling[poll.Name] = poll
}

func createSonicWallCpuUtilization() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.performance.cpuusagepercent"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.8714.2.1.3.1.1.0"))
	return attr
}

func createSonicWallMemoryPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("sonicWallMemory")
	poll.What = ".1.3.6.1.4.1.8714.2.1.3.1.2.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createSonicWallMemoryUtilization())
	p.Polling[poll.Name] = poll
}

func createSonicWallMemoryUtilization() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.performance.memoryusagepercent"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.8714.2.1.3.1.2.0"))
	return attr
}

func createSonicWallTemperaturePoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("sonicWallTemperature")
	poll.What = ".1.3.6.1.4.1.8714.2.1.3.1.4.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createSonicWallTemperature())
	p.Polling[poll.Name] = poll
}

func createSonicWallTemperature() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.temperature"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.8714.2.1.3.1.4.0"))
	return attr
}

// SonicWall-specific attribute creation functions
func createSonicWallVendor() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.vendor"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createContainsRule("sonicwall", ".1.3.6.1.2.1.1.1.0", "SonicWall"))
	return attr
}

func createSonicWallVersion() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.version"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.8714.2.1.1.1.0"))
	return attr
}

func createSonicWallSerialPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("sonicWallSerial")
	poll.What = ".1.3.6.1.4.1.8714.2.1.1.2.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createSonicWallSerial())
	p.Polling[poll.Name] = poll
}

func createSonicWallSerial() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.serialnumber"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.8714.2.1.1.2.0"))
	return attr
}
