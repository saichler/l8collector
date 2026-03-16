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

// CreateExtremeSwitchBootPolls creates collection and parsing Pollaris model for Extreme switches
func CreateExtremeSwitchBootPolls() *l8tpollaris.L8Pollaris {
	polaris := &l8tpollaris.L8Pollaris{}
	polaris.Name = "extreme-switch"
	polaris.Groups = []string{"extreme", "extreme-switch"}
	polaris.Polling = make(map[string]*l8tpollaris.L8Poll)
	createExtremeSystemPoll(polaris)
	createExtremeMibSystemPoll(polaris)
	createExtremeSerialPoll(polaris)
	createExtremeInterfacesPoll(polaris)
	createExtremeCpuPoll(polaris)
	createExtremeMemoryPoll(polaris)
	createExtremeTemperaturePoll(polaris)
	createOspfPoll(polaris, "extremeOspf")
	createBgpPoll(polaris, "extremeBgp")
	createVrfSshPoll(polaris, "extremeVrf", "show ip vrf detail", "voss")
	return polaris
}

// Extreme device-specific polling functions
func createExtremeSystemPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("extremeSystem")
	poll.What = ".1.3.6.1.4.1.1916.1.1.1.13.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createExtremeVersion())
	p.Polling[poll.Name] = poll
}

func createExtremeMibSystemPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("extremeMibSystem")
	poll.What = ".1.3.6.1.2.1.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createExtremeVendor())
	poll.Attributes = append(poll.Attributes, createSysName())
	p.Polling[poll.Name] = poll
}

func createExtremeInterfacesPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("extremeInterfaces")
	poll.What = ".1.3.6.1.2.1.2.2.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createInterfaceName())
	poll.Attributes = append(poll.Attributes, createInterfaceStatus())
	poll.Attributes = append(poll.Attributes, createInterfaceSpeed())
	p.Polling[poll.Name] = poll
}

func createExtremeCpuPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("extremeCpu")
	poll.What = ".1.3.6.1.4.1.1916.1.32.1.4.1.5.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createExtremeCpuUtilization())
	p.Polling[poll.Name] = poll
}

func createExtremeCpuUtilization() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.performance.cpuusagepercent"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.1916.1.32.1.4.1.5.1"))
	return attr
}

func createExtremeMemoryPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("extremeMemory")
	poll.What = ".1.3.6.1.4.1.1916.1.32.2.2.1.3.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createExtremeMemoryUtilization())
	p.Polling[poll.Name] = poll
}

func createExtremeMemoryUtilization() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.performance.memoryusagepercent"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.1916.1.32.2.2.1.3.1"))
	return attr
}

func createExtremeTemperaturePoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("extremeTemperature")
	poll.What = ".1.3.6.1.4.1.1916.1.1.1.8.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createExtremeTemperature())
	p.Polling[poll.Name] = poll
}

func createExtremeTemperature() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.temperature"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.1916.1.1.1.8.0"))
	return attr
}

// Extreme-specific attribute creation functions
func createExtremeVendor() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.vendor"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createContainsRule("extreme", ".1.3.6.1.2.1.1.1.0", "Extreme Networks"))
	return attr
}

func createExtremeVersion() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.version"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.1916.1.1.1.13.0"))
	return attr
}

func createExtremeSerialPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("extremeSerial")
	poll.What = ".1.3.6.1.4.1.1916.1.1.1.18.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createExtremeSerial())
	p.Polling[poll.Name] = poll
}

func createExtremeSerial() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.serialnumber"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.1916.1.1.1.18.0"))
	return attr
}
