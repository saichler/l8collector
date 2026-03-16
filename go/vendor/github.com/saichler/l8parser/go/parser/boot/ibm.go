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

// CreateIBMServerBootPolls creates collection and parsing Pollaris model for IBM servers
func CreateIBMServerBootPolls() *l8tpollaris.L8Pollaris {
	polaris := &l8tpollaris.L8Pollaris{}
	polaris.Name = "ibm-server"
	polaris.Groups = []string{"ibm", "ibm-server"}
	polaris.Polling = make(map[string]*l8tpollaris.L8Poll)
	createIBMSystemPoll(polaris)
	createIBMMibSystemPoll(polaris)
	createIBMSerialPoll(polaris)
	createIBMStoragePoll(polaris)
	createIBMCpuPoll(polaris)
	createIBMMemoryPoll(polaris)
	createIBMTemperaturePoll(polaris)
	return polaris
}

// IBM server-specific polling functions
func createIBMSystemPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("ibmSystem")
	poll.What = ".1.3.6.1.4.1.2.6.220.2.1.1.1.1.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createIBMVersion())
	p.Polling[poll.Name] = poll
}

func createIBMMibSystemPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("ibmMibSystem")
	poll.What = ".1.3.6.1.2.1.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createIBMVendor())
	poll.Attributes = append(poll.Attributes, createSysName())
	p.Polling[poll.Name] = poll
}

func createIBMStoragePoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("ibmStorage")
	poll.What = ".1.3.6.1.2.1.25.2.3.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createDiskStatus())
	p.Polling[poll.Name] = poll
}

func createIBMCpuPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("ibmCpu")
	poll.What = ".1.3.6.1.4.1.2.6.220.2.1.1.1.5.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createIBMCpuUtilization())
	p.Polling[poll.Name] = poll
}

func createIBMCpuUtilization() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.performance.cpuusagepercent"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.2.6.220.2.1.1.1.5.0"))
	return attr
}

func createIBMMemoryPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("ibmMemory")
	poll.What = ".1.3.6.1.4.1.2.6.220.2.1.2.1.5.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createIBMMemoryUtilization())
	p.Polling[poll.Name] = poll
}

func createIBMMemoryUtilization() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.performance.memoryusagepercent"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.2.6.220.2.1.2.1.5.0"))
	return attr
}

func createIBMTemperaturePoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("ibmTemperature")
	poll.What = ".1.3.6.1.4.1.2.6.220.2.1.3.1.4.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createIBMTemperature())
	p.Polling[poll.Name] = poll
}

func createIBMTemperature() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.temperature"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.2.6.220.2.1.3.1.4.0"))
	return attr
}

// IBM-specific attribute creation functions
func createIBMVendor() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.vendor"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createContainsRule("ibm", ".1.3.6.1.2.1.1.1.0", "IBM"))
	return attr
}

func createIBMVersion() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.version"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.2.6.220.2.1.1.1.1.0"))
	return attr
}

func createIBMSerialPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("ibmSerial")
	poll.What = ".1.3.6.1.4.1.2.6.220.2.1.1.1.2.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createIBMSerial())
	p.Polling[poll.Name] = poll
}

func createIBMSerial() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.serialnumber"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.2.6.220.2.1.1.1.2.0"))
	return attr
}
