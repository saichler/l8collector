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

// CreateHPEServerBootPolls creates collection and parsing Pollaris model for HPE servers
func CreateHPEServerBootPolls() *l8tpollaris.L8Pollaris {
	polaris := &l8tpollaris.L8Pollaris{}
	polaris.Name = "hpe-server"
	polaris.Groups = []string{"hpe", "hpe-server"}
	polaris.Polling = make(map[string]*l8tpollaris.L8Poll)
	createHPESystemPoll(polaris)
	createHPEMibSystemPoll(polaris)
	createHPESerialPoll(polaris)
	createHPEStoragePoll(polaris)
	createHPECpuPoll(polaris)
	createHPEMemoryPoll(polaris)
	createHPETemperaturePoll(polaris)
	return polaris
}

// HPE server-specific polling functions
func createHPESystemPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("hpeSystem")
	poll.What = ".1.3.6.1.4.1.232.2.2.4.2.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createHPEVersion())
	p.Polling[poll.Name] = poll
}

func createHPEMibSystemPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("hpeMibSystem")
	poll.What = ".1.3.6.1.2.1.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createHPEVendor())
	poll.Attributes = append(poll.Attributes, createSysName())
	p.Polling[poll.Name] = poll
}

func createHPEStoragePoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("hpeStorage")
	poll.What = ".1.3.6.1.2.1.25.2.3.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createDiskStatus())
	p.Polling[poll.Name] = poll
}

func createHPECpuPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("hpeCpu")
	poll.What = ".1.3.6.1.4.1.232.11.2.3.1.1.3.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createHPECpuUtilization())
	p.Polling[poll.Name] = poll
}

func createHPECpuUtilization() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.performance.cpuusagepercent"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.232.11.2.3.1.1.3.0"))
	return attr
}

func createHPEMemoryPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("hpeMemory")
	poll.What = ".1.3.6.1.4.1.232.11.2.13.1.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createHPEMemoryUtilization())
	p.Polling[poll.Name] = poll
}

func createHPEMemoryUtilization() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.performance.memoryusagepercent"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.232.11.2.13.1.0"))
	return attr
}

func createHPETemperaturePoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("hpeTemperature")
	poll.What = ".1.3.6.1.4.1.232.6.2.6.8.1.4.0.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createHPETemperature())
	p.Polling[poll.Name] = poll
}

func createHPETemperature() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.temperature"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.232.6.2.6.8.1.4.0.1"))
	return attr
}

// HPE-specific attribute creation functions
func createHPEVendor() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.vendor"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createContainsRule("hpe", ".1.3.6.1.2.1.1.1.0", "Hewlett Packard Enterprise"))
	return attr
}

func createHPEVersion() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.version"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.232.2.2.4.2.0"))
	return attr
}

func createHPESerialPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("hpeSerial")
	poll.What = ".1.3.6.1.4.1.232.2.2.2.1.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createHPESerial())
	p.Polling[poll.Name] = poll
}

func createHPESerial() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.serialnumber"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.232.2.2.2.1.0")) // cpqSiSysSerialNum
	return attr
}
