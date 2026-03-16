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

// createOspfPoll creates an OSPF MIB polling configuration (standard MIB-II, same across all vendors).
// Walks the entire OSPF MIB subtree (.1.3.6.1.2.1.14) to collect general params, area table,
// interface table, and neighbor table.
func createOspfPoll(p *l8tpollaris.L8Pollaris, pollName string) {
	poll := createBaseSNMPPoll(pollName)
	poll.What = ".1.3.6.1.2.1.14"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createOspfAttribute())
	p.Polling[poll.Name] = poll
}

// createOspfAttribute creates the attribute that maps the OSPF MIB walk result
// to networkdevice.logicals.vrfs.ospfinfo using the SnmpOspfToVrf bulk rule.
func createOspfAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.logicals.vrfs.ospfinfo"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	rule := &l8tpollaris.L8PRule{}
	rule.Name = "SnmpOspfToVrf"
	rule.Params = make(map[string]*l8tpollaris.L8PParameter)
	attr.Rules = append(attr.Rules, rule)
	return attr
}

// createBgpPoll creates a BGP4 MIB polling configuration (standard MIB-II, same across all vendors).
// Walks the entire BGP4 MIB subtree (.1.3.6.1.2.1.15) to collect global params and peer table.
func createBgpPoll(p *l8tpollaris.L8Pollaris, pollName string) {
	poll := createBaseSNMPPoll(pollName)
	poll.What = ".1.3.6.1.2.1.15"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createBgpAttribute())
	p.Polling[poll.Name] = poll
}

// createBgpAttribute creates the attribute that maps the BGP4 MIB walk result
// to networkdevice.logicals.vrfs.bgpinfo using the SnmpBgpToVrf bulk rule.
func createBgpAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.logicals.vrfs.bgpinfo"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	rule := &l8tpollaris.L8PRule{}
	rule.Name = "SnmpBgpToVrf"
	rule.Params = make(map[string]*l8tpollaris.L8PParameter)
	attr.Rules = append(attr.Rules, rule)
	return attr
}

// createVrfSshPoll creates a VRF polling configuration via SSH.
// VRF has no standard SNMP OIDs; it is polled via vendor-specific SSH commands.
// The format parameter identifies which vendor-specific parser to use.
func createVrfSshPoll(p *l8tpollaris.L8Pollaris, pollName, sshCommand, format string) {
	poll := &l8tpollaris.L8Poll{}
	poll.Name = pollName
	poll.What = sshCommand
	poll.Protocol = l8tpollaris.L8PProtocol_L8PSSH
	poll.Cadence = DEFAULT_CADENCE
	poll.Timeout = DEFAULT_TIMEOUT
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createVrfSshAttribute(format))
	p.Polling[poll.Name] = poll
}

// createVrfSshAttribute creates the attribute that maps SSH VRF command output
// to networkdevice.logicals.vrfs using the SshVrfParse rule.
// The format parameter specifies the vendor output format (iosxr, ios, nxos, junos, timos, vrp, eos, voss, univerge).
func createVrfSshAttribute(format string) *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.logicals.vrfs"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	rule := &l8tpollaris.L8PRule{}
	rule.Name = "SshVrfParse"
	rule.Params = make(map[string]*l8tpollaris.L8PParameter)
	addParameter("format", format, rule)
	attr.Rules = append(attr.Rules, rule)
	return attr
}
