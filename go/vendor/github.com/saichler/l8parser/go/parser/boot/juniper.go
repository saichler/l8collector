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

// CreateJuniperRouterBootPolls creates collection and parsing Pollaris model for Juniper routers
func CreateJuniperRouterBootPolls() *l8tpollaris.L8Pollaris {
	polaris := &l8tpollaris.L8Pollaris{}
	polaris.Name = "juniper-router"
	polaris.Groups = []string{"juniper", "juniper-router"}
	polaris.Polling = make(map[string]*l8tpollaris.L8Poll)
	createJuniperSystemPoll(polaris)
	createJuniperMibSystemPoll(polaris)
	createJuniperSerialPoll(polaris)
	createJuniperFirmwarePoll(polaris)
	createJuniperInterfacesPoll(polaris)
	createJuniperCpuPoll(polaris)
	createJuniperMemoryPoll(polaris)
	createJuniperTemperaturePoll(polaris)
	createOspfPoll(polaris, "juniperOspf")
	createBgpPoll(polaris, "juniperBgp")
	createVrfSshPoll(polaris, "juniperVrf", "show route instance detail", "junos")
	return polaris
}

// Juniper device-specific polling functions
func createJuniperSystemPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("juniperSystem")
	poll.What = ".1.3.6.1.4.1.2636.3.1.2.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createJuniperVersion())
	p.Polling[poll.Name] = poll
}

func createJuniperMibSystemPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("juniperMibSystem")
	poll.What = ".1.3.6.1.2.1.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createJuniperVendor())
	poll.Attributes = append(poll.Attributes, createSysName())
	p.Polling[poll.Name] = poll
}

func createJuniperInterfacesPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("juniperInterfaces")
	poll.What = ".1.3.6.1.2.1.2.2.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createInterfaceName())
	poll.Attributes = append(poll.Attributes, createInterfaceStatus())
	poll.Attributes = append(poll.Attributes, createInterfaceSpeed())
	p.Polling[poll.Name] = poll
}

func createJuniperCpuPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("juniperCpu")
	poll.What = ".1.3.6.1.4.1.2636.3.1.13.1.8.9.1.0.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createJuniperCpuUtilization())
	p.Polling[poll.Name] = poll
}

func createJuniperCpuUtilization() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.performance.cpuusagepercent"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.2636.3.1.13.1.8.9.1.0.0"))
	return attr
}

func createJuniperMemoryPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("juniperMemory")
	poll.What = ".1.3.6.1.4.1.2636.3.1.13.1.11.9.1.0.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createJuniperMemoryUtilization())
	p.Polling[poll.Name] = poll
}

func createJuniperMemoryUtilization() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.performance.memoryusagepercent"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.2636.3.1.13.1.11.9.1.0.0"))
	return attr
}

func createJuniperTemperaturePoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("juniperTemperature")
	poll.What = ".1.3.6.1.4.1.2636.3.1.13.1.7.9.1.0.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createJuniperTemperature())
	p.Polling[poll.Name] = poll
}

func createJuniperTemperature() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.temperature"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.2636.3.1.13.1.7.9.1.0.0"))
	return attr
}

// Juniper-specific attribute creation functions
func createJuniperVendor() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.vendor"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createContainsRule("juniper", ".1.3.6.1.2.1.1.1.0", "Juniper"))
	return attr
}

func createJuniperVersion() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.version"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.2636.3.1.2.0"))
	return attr
}

func createJuniperSerialPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("juniperSerial")
	poll.What = ".1.3.6.1.4.1.2636.3.1.3.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createJuniperSerial())
	p.Polling[poll.Name] = poll
}

func createJuniperSerial() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.serialnumber"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.2636.3.1.3.0")) // jnxBoxSerialNo
	return attr
}

func createJuniperFirmwarePoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("juniperFirmware")
	poll.What = ".1.3.6.1.4.1.2636.3.40.1.4.1.1.1.5"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createJuniperFirmwareVersion())
	p.Polling[poll.Name] = poll
}

func createJuniperFirmwareVersion() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.firmwareversion"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.2636.3.40.1.4.1.1.1.5")) // jnxFWDetectorVersion
	return attr
}
