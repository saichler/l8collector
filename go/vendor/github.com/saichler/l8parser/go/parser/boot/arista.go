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

// CreateAristaSwitchBootPolls creates collection and parsing Pollaris model for Arista switches
func CreateAristaSwitchBootPolls() *l8tpollaris.L8Pollaris {
	polaris := &l8tpollaris.L8Pollaris{}
	polaris.Name = "arista-switch"
	polaris.Groups = []string{"arista", "arista-switch"}
	polaris.Polling = make(map[string]*l8tpollaris.L8Poll)
	createAristaSystemPoll(polaris)
	createAristaMibSystemPoll(polaris)
	createAristaSerialPoll(polaris)
	createAristaFirmwarePoll(polaris)
	createAristaInterfacesPoll(polaris)
	createAristaCpuPoll(polaris)
	createAristaMemoryPoll(polaris)
	createAristaTemperaturePoll(polaris)
	createOspfPoll(polaris, "aristaOspf")
	createBgpPoll(polaris, "aristaBgp")
	createVrfSshPoll(polaris, "aristaVrf", "show vrf detail", "eos")
	return polaris
}

// Arista device-specific polling functions
func createAristaSystemPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("aristaSystem")
	poll.What = ".1.3.6.1.4.1.30065.1.3.1.1.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createAristaVersion())
	p.Polling[poll.Name] = poll
}

func createAristaMibSystemPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("aristaMibSystem")
	poll.What = ".1.3.6.1.2.1.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createAristaVendor())
	poll.Attributes = append(poll.Attributes, createSysName())
	p.Polling[poll.Name] = poll
}

func createAristaInterfacesPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("aristaInterfaces")
	poll.What = ".1.3.6.1.2.1.2.2.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createInterfaceName())
	poll.Attributes = append(poll.Attributes, createInterfaceStatus())
	poll.Attributes = append(poll.Attributes, createInterfaceSpeed())
	p.Polling[poll.Name] = poll
}

func createAristaCpuPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("aristaCpu")
	poll.What = ".1.3.6.1.2.1.25.3.3.1.2.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createAristaCpuUtilization())
	p.Polling[poll.Name] = poll
}

func createAristaCpuUtilization() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.performance.cpuusagepercent"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.2.1.25.3.3.1.2.1"))
	return attr
}

func createAristaMemoryPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("aristaMemory")
	poll.What = ".1.3.6.1.2.1.25.2.3.1.6.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createAristaMemoryUtilization())
	p.Polling[poll.Name] = poll
}

func createAristaMemoryUtilization() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.performance.memoryusagepercent"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.2.1.25.2.3.1.6.1"))
	return attr
}

func createAristaTemperaturePoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("aristaTemperature")
	poll.What = ".1.3.6.1.2.1.99.1.1.1.4.100006"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createAristaTemperature())
	p.Polling[poll.Name] = poll
}

func createAristaTemperature() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.temperature"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.2.1.99.1.1.1.4.100006"))
	return attr
}

// Arista-specific attribute creation functions
func createAristaVendor() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.vendor"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createContainsRule("arista", ".1.3.6.1.2.1.1.1.0", "Arista"))
	return attr
}

func createAristaVersion() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.version"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.30065.1.3.1.1.0"))
	return attr
}

func createAristaSerialPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("aristaSerial")
	poll.What = ".1.3.6.1.2.1.47.1.1.1.1.11.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createAristaSerial())
	p.Polling[poll.Name] = poll
}

func createAristaSerial() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.serialnumber"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.11.1")) // entPhysicalSerialNum (Arista supports standard Entity MIB)
	return attr
}

func createAristaFirmwarePoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("aristaFirmware")
	poll.What = ".1.3.6.1.4.1.30065.1.3.1.1.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createAristaFirmwareVersion())
	p.Polling[poll.Name] = poll
}

func createAristaFirmwareVersion() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.firmwareversion"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.30065.1.3.1.1.0")) // ARISTA-GENERAL-MIB
	return attr
}
