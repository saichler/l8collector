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

// CreateHuaweiRouterBootPolls creates collection and parsing Pollaris model for Huawei routers
func CreateHuaweiRouterBootPolls() *l8tpollaris.L8Pollaris {
	polaris := &l8tpollaris.L8Pollaris{}
	polaris.Name = "huawei-router"
	polaris.Groups = []string{"huawei", "huawei-router"}
	polaris.Polling = make(map[string]*l8tpollaris.L8Poll)
	createHuaweiSystemPoll(polaris)
	createHuaweiMibSystemPoll(polaris)
	createHuaweiSerialPoll(polaris)
	createHuaweiInterfacesPoll(polaris)
	createHuaweiCpuPoll(polaris)
	createHuaweiMemoryPoll(polaris)
	createHuaweiTemperaturePoll(polaris)
	createOspfPoll(polaris, "huaweiOspf")
	createBgpPoll(polaris, "huaweiBgp")
	createVrfSshPoll(polaris, "huaweiVrf", "display ip vpn-instance verbose", "vrp")
	return polaris
}

// Huawei device-specific polling functions
func createHuaweiSystemPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("huaweiSystem")
	poll.What = ".1.3.6.1.4.1.2011.5.25.1.1.1.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createHuaweiVersion())
	p.Polling[poll.Name] = poll
}

func createHuaweiMibSystemPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("huaweiMibSystem")
	poll.What = ".1.3.6.1.2.1.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createHuaweiVendor())
	poll.Attributes = append(poll.Attributes, createSysName())
	p.Polling[poll.Name] = poll
}

func createHuaweiInterfacesPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("huaweiInterfaces")
	poll.What = ".1.3.6.1.2.1.2.2.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createInterfaceName())
	poll.Attributes = append(poll.Attributes, createInterfaceStatus())
	p.Polling[poll.Name] = poll
}

func createHuaweiCpuPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("huaweiCpu")
	poll.What = ".1.3.6.1.4.1.2011.5.25.31.1.1.1.1.5.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createHuaweiCpuUtilization())
	p.Polling[poll.Name] = poll
}

func createHuaweiCpuUtilization() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.performance.cpuusagepercent"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.2011.5.25.31.1.1.1.1.5.0"))
	return attr
}

func createHuaweiMemoryPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("huaweiMemory")
	poll.What = ".1.3.6.1.4.1.2011.5.25.31.1.1.1.1.7.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createHuaweiMemoryUtilization())
	p.Polling[poll.Name] = poll
}

func createHuaweiMemoryUtilization() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.performance.memoryusagepercent"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.2011.5.25.31.1.1.1.1.7.0"))
	return attr
}

func createHuaweiTemperaturePoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("huaweiTemperature")
	poll.What = ".1.3.6.1.4.1.2011.5.25.31.1.1.1.1.11.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createHuaweiTemperature())
	p.Polling[poll.Name] = poll
}

func createHuaweiTemperature() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.temperature"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.2011.5.25.31.1.1.1.1.11.0"))
	return attr
}

// Huawei-specific attribute creation functions
func createHuaweiVendor() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.vendor"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createContainsRule("huawei", ".1.3.6.1.2.1.1.1.0", "Huawei"))
	return attr
}

func createHuaweiVersion() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.version"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.2011.5.25.1.1.1.0"))
	return attr
}

func createHuaweiSerialPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("huaweiSerial")
	poll.What = ".1.3.6.1.4.1.2011.5.25.31.1.1.1.1.15.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createHuaweiSerial())
	p.Polling[poll.Name] = poll
}

func createHuaweiSerial() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.serialnumber"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.2011.5.25.31.1.1.1.1.15.1")) // HUAWEI-ENTITY-EXTENT-MIB
	return attr
}
