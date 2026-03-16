/*
 * © 2025 Sharon Aicler (saichler@gmail.com)
 *
 * Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
 * You may obtain a copy of the License at:
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package creates

import (
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
)

func CreateDevice(ip, linksId, crId string) *l8tpollaris.L8PTarget {
	device := &l8tpollaris.L8PTarget{}
	device.TargetId = ip
	device.LinksId = linksId
	device.Hosts = make(map[string]*l8tpollaris.L8PHost)
	device.InventoryType = l8tpollaris.L8PTargetType_Network_Device
	device.State = l8tpollaris.L8PTargetState_Down
	host := &l8tpollaris.L8PHost{}
	host.HostId = ip

	host.Configs = make(map[int32]*l8tpollaris.L8PHostProtocol)
	device.Hosts[host.HostId] = host

	sshConfig := &l8tpollaris.L8PHostProtocol{}
	sshConfig.Protocol = l8tpollaris.L8PProtocol_L8PSSH
	sshConfig.Port = 22
	sshConfig.Addr = ip
	sshConfig.CredId = crId
	sshConfig.Terminal = "vt100"
	sshConfig.Timeout = 60

	host.Configs[int32(sshConfig.Protocol)] = sshConfig

	snmpConfig := &l8tpollaris.L8PHostProtocol{}
	snmpConfig.Protocol = l8tpollaris.L8PProtocol_L8PPSNMPV2
	snmpConfig.Addr = ip
	snmpConfig.Port = 161
	snmpConfig.Timeout = 60
	snmpConfig.CredId = crId

	host.Configs[int32(snmpConfig.Protocol)] = snmpConfig

	return device
}
