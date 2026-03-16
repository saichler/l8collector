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

// CreateNECRouterBootPolls creates collection and parsing Pollaris model for NEC routers
func CreateNECRouterBootPolls() *l8tpollaris.L8Pollaris {
	polaris := &l8tpollaris.L8Pollaris{}
	polaris.Name = "nec-router"
	polaris.Groups = []string{"nec", "nec-router"}
	polaris.Polling = make(map[string]*l8tpollaris.L8Poll)
	createNECSystemPoll(polaris)
	createNECMibSystemPoll(polaris)
	createNECSerialPoll(polaris)
	createNECInterfacesPoll(polaris)
	createNECCpuPoll(polaris)
	createNECMemoryPoll(polaris)
	createNECTemperaturePoll(polaris)
	createOspfPoll(polaris, "necOspf")
	createBgpPoll(polaris, "necBgp")
	createVrfSshPoll(polaris, "necVrf", "show ip vrf detail", "univerge")
	return polaris
}

// NEC device-specific polling functions
func createNECSystemPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("necSystem")
	poll.What = ".1.3.6.1.4.1.119.2.3.84.1.1.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createNECVersion())
	p.Polling[poll.Name] = poll
}

func createNECMibSystemPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("necMibSystem")
	poll.What = ".1.3.6.1.2.1.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createNECVendor())
	poll.Attributes = append(poll.Attributes, createSysName())
	p.Polling[poll.Name] = poll
}

func createNECInterfacesPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("necInterfaces")
	poll.What = ".1.3.6.1.2.1.2.2.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createInterfaceName())
	poll.Attributes = append(poll.Attributes, createInterfaceStatus())
	p.Polling[poll.Name] = poll
}

func createNECCpuPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("necCpu")
	poll.What = ".1.3.6.1.4.1.119.2.3.84.3.1.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createNECCpuUtilization())
	p.Polling[poll.Name] = poll
}

func createNECCpuUtilization() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.performance.cpuusagepercent"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.119.2.3.84.3.1.0"))
	return attr
}

func createNECMemoryPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("necMemory")
	poll.What = ".1.3.6.1.4.1.119.2.3.84.3.3.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createNECMemoryUtilization())
	p.Polling[poll.Name] = poll
}

func createNECMemoryUtilization() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.performance.memoryusagepercent"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.119.2.3.84.3.3.0"))
	return attr
}

func createNECTemperaturePoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("necTemperature")
	poll.What = ".1.3.6.1.4.1.119.2.3.84.3.5.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createNECTemperature())
	p.Polling[poll.Name] = poll
}

func createNECTemperature() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.temperature"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.119.2.3.84.3.5.0"))
	return attr
}

// NEC-specific attribute creation functions
func createNECVendor() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.vendor"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createContainsRule("nec", ".1.3.6.1.2.1.1.1.0", "NEC"))
	return attr
}

func createNECVersion() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.version"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.119.2.3.84.1.1.0"))
	return attr
}

func createNECSerialPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("necSerial")
	poll.What = ".1.3.6.1.4.1.119.2.3.84.1.2.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createNECSerial())
	p.Polling[poll.Name] = poll
}

func createNECSerial() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.serialnumber"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.119.2.3.84.1.2.0"))
	return attr
}
