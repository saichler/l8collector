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

// CreateNokiaRouterBootPolls creates collection and parsing Pollaris model for Nokia routers
func CreateNokiaRouterBootPolls() *l8tpollaris.L8Pollaris {
	polaris := &l8tpollaris.L8Pollaris{}
	polaris.Name = "nokia-router"
	polaris.Groups = []string{"nokia", "nokia-router"}
	polaris.Polling = make(map[string]*l8tpollaris.L8Poll)
	createNokiaSystemPoll(polaris)
	createNokiaMibSystemPoll(polaris)
	createNokiaSerialPoll(polaris)
	createNokiaInterfacesPoll(polaris)
	createNokiaCpuPoll(polaris)
	createNokiaMemoryPoll(polaris)
	createNokiaTemperaturePoll(polaris)
	createOspfPoll(polaris, "nokiaOspf")
	createBgpPoll(polaris, "nokiaBgp")
	createVrfSshPoll(polaris, "nokiaVrf", "show router vrf", "timos")
	return polaris
}

// Nokia device-specific polling functions
func createNokiaSystemPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("nokiaSystem")
	poll.What = ".1.3.6.1.4.1.6527.3.1.2.2.1.4.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createNokiaVersion())
	p.Polling[poll.Name] = poll
}

func createNokiaMibSystemPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("nokiaMibSystem")
	poll.What = ".1.3.6.1.2.1.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createNokiaVendor())
	poll.Attributes = append(poll.Attributes, createSysName())
	p.Polling[poll.Name] = poll
}

func createNokiaInterfacesPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("nokiaInterfaces")
	poll.What = ".1.3.6.1.2.1.2.2.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createInterfaceName())
	poll.Attributes = append(poll.Attributes, createInterfaceStatus())
	p.Polling[poll.Name] = poll
}

func createNokiaCpuPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("nokiaCpu")
	poll.What = ".1.3.6.1.4.1.6527.3.1.2.1.1.1.1.5.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createNokiaCpuUtilization())
	p.Polling[poll.Name] = poll
}

func createNokiaCpuUtilization() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.performance.cpuusagepercent"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.6527.3.1.2.1.1.1.1.5.1"))
	return attr
}

func createNokiaMemoryPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("nokiaMemory")
	poll.What = ".1.3.6.1.4.1.6527.3.1.2.1.1.1.1.10.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createNokiaMemoryUtilization())
	p.Polling[poll.Name] = poll
}

func createNokiaMemoryUtilization() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.performance.memoryusagepercent"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.6527.3.1.2.1.1.1.1.10.1"))
	return attr
}

func createNokiaTemperaturePoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("nokiaTemperature")
	poll.What = ".1.3.6.1.4.1.6527.3.1.2.1.1.1.1.7.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createNokiaTemperature())
	p.Polling[poll.Name] = poll
}

func createNokiaTemperature() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.temperature"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.6527.3.1.2.1.1.1.1.7.1"))
	return attr
}

// Nokia-specific attribute creation functions
func createNokiaVendor() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.vendor"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createContainsRule("nokia", ".1.3.6.1.2.1.1.1.0", "Nokia"))
	return attr
}

func createNokiaVersion() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.version"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.6527.3.1.2.2.1.4.0"))
	return attr
}

func createNokiaSerialPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("nokiaSerial")
	poll.What = ".1.3.6.1.4.1.6527.3.1.2.2.1.6.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createNokiaSerial())
	p.Polling[poll.Name] = poll
}

func createNokiaSerial() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.serialnumber"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.6527.3.1.2.2.1.6.0")) // TIMETRA-CHASSIS-MIB
	return attr
}
