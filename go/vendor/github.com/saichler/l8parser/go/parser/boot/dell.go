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

// CreateDellServerBootPolls creates collection and parsing Pollaris model for Dell servers
func CreateDellServerBootPolls() *l8tpollaris.L8Pollaris {
	polaris := &l8tpollaris.L8Pollaris{}
	polaris.Name = "dell-server"
	polaris.Groups = []string{"dell", "dell-server"}
	polaris.Polling = make(map[string]*l8tpollaris.L8Poll)
	createDellSystemPoll(polaris)
	createDellMibSystemPoll(polaris)
	createDellSerialPoll(polaris)
	createDellStoragePoll(polaris)
	createDellCpuPoll(polaris)
	createDellMemoryPoll(polaris)
	createDellTemperaturePoll(polaris)
	return polaris
}

// Dell server-specific polling functions
func createDellSystemPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("dellSystem")
	poll.What = ".1.3.6.1.4.1.674.10892.5.1.3.1.6.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createDellVersion())
	p.Polling[poll.Name] = poll
}

func createDellMibSystemPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("dellMibSystem")
	poll.What = ".1.3.6.1.2.1.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createDellVendor())
	poll.Attributes = append(poll.Attributes, createSysName())
	p.Polling[poll.Name] = poll
}

func createDellStoragePoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("dellStorage")
	poll.What = ".1.3.6.1.2.1.25.2.3.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createDiskStatus())
	p.Polling[poll.Name] = poll
}

func createDellCpuPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("dellCpu")
	poll.What = ".1.3.6.1.4.1.674.10892.5.4.200.10.1.12.1.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createDellCpuUtilization())
	p.Polling[poll.Name] = poll
}

func createDellCpuUtilization() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.performance.cpuusagepercent"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.674.10892.5.4.200.10.1.12.1.1"))
	return attr
}

func createDellMemoryPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("dellMemory")
	poll.What = ".1.3.6.1.2.1.25.2.3.1.6.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createDellMemoryUtilization())
	p.Polling[poll.Name] = poll
}

func createDellMemoryUtilization() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.performance.memoryusagepercent"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.2.1.25.2.3.1.6.1"))
	return attr
}

func createDellTemperaturePoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("dellTemperature")
	poll.What = ".1.3.6.1.4.1.674.10892.5.4.700.20.1.6.1.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createDellTemperature())
	p.Polling[poll.Name] = poll
}

func createDellTemperature() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.temperature"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.674.10892.5.4.700.20.1.6.1.1"))
	return attr
}

// Dell-specific attribute creation functions
func createDellVendor() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.vendor"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createContainsRule("dell", ".1.3.6.1.2.1.1.1.0", "Dell"))
	return attr
}

func createDellVersion() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.version"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.674.10892.5.1.3.1.6.0"))
	return attr
}

func createDellSerialPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("dellSerial")
	poll.What = ".1.3.6.1.4.1.674.10892.5.1.3.2.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createDellSerial())
	p.Polling[poll.Name] = poll
}

func createDellSerial() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.serialnumber"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.674.10892.5.1.3.2.0")) // systemServiceTag
	return attr
}
