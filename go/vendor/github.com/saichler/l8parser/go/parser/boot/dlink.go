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

// CreateDLinkSwitchBootPolls creates collection and parsing Pollaris model for D-Link switches
func CreateDLinkSwitchBootPolls() *l8tpollaris.L8Pollaris {
	polaris := &l8tpollaris.L8Pollaris{}
	polaris.Name = "dlink-switch"
	polaris.Groups = []string{"dlink", "dlink-switch"}
	polaris.Polling = make(map[string]*l8tpollaris.L8Poll)
	createDLinkSystemPoll(polaris)
	createDLinkMibSystemPoll(polaris)
	createDLinkSerialPoll(polaris)
	createDLinkInterfacesPoll(polaris)
	createDLinkCpuPoll(polaris)
	createDLinkMemoryPoll(polaris)
	createDLinkTemperaturePoll(polaris)
	return polaris
}

// D-Link device-specific polling functions
func createDLinkSystemPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("dLinkSystem")
	poll.What = ".1.3.6.1.4.1.171.12.1.1.1.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createDLinkVersion())
	p.Polling[poll.Name] = poll
}

func createDLinkMibSystemPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("dLinkMibSystem")
	poll.What = ".1.3.6.1.2.1.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createDLinkVendor())
	poll.Attributes = append(poll.Attributes, createSysName())
	p.Polling[poll.Name] = poll
}

func createDLinkInterfacesPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("dLinkInterfaces")
	poll.What = ".1.3.6.1.2.1.2.2.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createInterfaceName())
	poll.Attributes = append(poll.Attributes, createInterfaceStatus())
	poll.Attributes = append(poll.Attributes, createInterfaceSpeed())
	p.Polling[poll.Name] = poll
}

func createDLinkCpuPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("dLinkCpu")
	poll.What = ".1.3.6.1.4.1.171.12.1.1.6.1.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createDLinkCpuUtilization())
	p.Polling[poll.Name] = poll
}

func createDLinkCpuUtilization() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.performance.cpuusagepercent"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.171.12.1.1.6.1.0"))
	return attr
}

func createDLinkMemoryPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("dLinkMemory")
	poll.What = ".1.3.6.1.4.1.171.12.1.1.9.4.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createDLinkMemoryUtilization())
	p.Polling[poll.Name] = poll
}

func createDLinkMemoryUtilization() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.performance.memoryusagepercent"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.171.12.1.1.9.4.0"))
	return attr
}

func createDLinkTemperaturePoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("dLinkTemperature")
	poll.What = ".1.3.6.1.4.1.171.12.11.1.1.6.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createDLinkTemperature())
	p.Polling[poll.Name] = poll
}

func createDLinkTemperature() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.temperature"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.171.12.11.1.1.6.1"))
	return attr
}

// D-Link-specific attribute creation functions
func createDLinkVendor() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.vendor"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createContainsRule("dlink", ".1.3.6.1.2.1.1.1.0", "D-Link"))
	return attr
}

func createDLinkVersion() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.version"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.171.12.1.1.1.0"))
	return attr
}

func createDLinkSerialPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("dLinkSerial")
	poll.What = ".1.3.6.1.4.1.171.12.1.1.12.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createDLinkSerial())
	p.Polling[poll.Name] = poll
}

func createDLinkSerial() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.serialnumber"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.171.12.1.1.12.0"))
	return attr
}
