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

// CreateCheckPointFirewallBootPolls creates collection and parsing Pollaris model for Check Point firewalls
func CreateCheckPointFirewallBootPolls() *l8tpollaris.L8Pollaris {
	polaris := &l8tpollaris.L8Pollaris{}
	polaris.Name = "checkpoint-firewall"
	polaris.Groups = []string{"checkpoint", "checkpoint-firewall"}
	polaris.Polling = make(map[string]*l8tpollaris.L8Poll)
	createCheckPointSystemPoll(polaris)
	createCheckPointMibSystemPoll(polaris)
	createCheckPointSerialPoll(polaris)
	createCheckPointInterfacesPoll(polaris)
	createCheckPointCpuPoll(polaris)
	createCheckPointMemoryPoll(polaris)
	createCheckPointTemperaturePoll(polaris)
	return polaris
}

// Check Point device-specific polling functions
func createCheckPointSystemPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("checkPointSystem")
	poll.What = ".1.3.6.1.4.1.2620.1.6.4.1.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createCheckPointVersion())
	p.Polling[poll.Name] = poll
}

func createCheckPointMibSystemPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("checkPointMibSystem")
	poll.What = ".1.3.6.1.2.1.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createCheckPointVendor())
	poll.Attributes = append(poll.Attributes, createSysName())
	p.Polling[poll.Name] = poll
}

func createCheckPointInterfacesPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("checkPointInterfaces")
	poll.What = ".1.3.6.1.2.1.2.2.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createInterfaceName())
	poll.Attributes = append(poll.Attributes, createInterfaceStatus())
	p.Polling[poll.Name] = poll
}

func createCheckPointCpuPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("checkPointCpu")
	poll.What = ".1.3.6.1.4.1.2620.1.6.7.2.7.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createCheckPointCpuUtilization())
	p.Polling[poll.Name] = poll
}

func createCheckPointCpuUtilization() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.performance.cpuusagepercent"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.2620.1.6.7.2.7.0"))
	return attr
}

func createCheckPointMemoryPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("checkPointMemory")
	poll.What = ".1.3.6.1.4.1.2620.1.6.7.4.3.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createCheckPointMemoryUtilization())
	p.Polling[poll.Name] = poll
}

func createCheckPointMemoryUtilization() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.performance.memoryusagepercent"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.2620.1.6.7.4.3.0"))
	return attr
}

func createCheckPointTemperaturePoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("checkPointTemperature")
	poll.What = ".1.3.6.1.4.1.2620.1.6.7.8.1.1.3.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createCheckPointTemperature())
	p.Polling[poll.Name] = poll
}

func createCheckPointTemperature() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.temperature"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.2620.1.6.7.8.1.1.3.0"))
	return attr
}

// Check Point-specific attribute creation functions
func createCheckPointVendor() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.vendor"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createContainsRule("checkpoint", ".1.3.6.1.2.1.1.1.0", "Check Point"))
	return attr
}

func createCheckPointVersion() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.version"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.2620.1.6.4.1.0"))
	return attr
}

func createCheckPointSerialPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("checkPointSerial")
	poll.What = ".1.3.6.1.4.1.2620.1.6.1.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createCheckPointSerial())
	p.Polling[poll.Name] = poll
}

func createCheckPointSerial() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.serialnumber"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.2620.1.6.1.0"))
	return attr
}
