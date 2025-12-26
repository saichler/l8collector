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

package service

import (
	"github.com/saichler/l8collector/go/collector/common"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8srlz/go/serialize/object"
)

// staticJobs is a registry of built-in jobs that are handled specially
// during the boot sequence. These jobs generate data from the collector's
// internal state rather than from protocol operations.
var staticJobs = map[string]StaticJob{(&IpAddressJob{}).what(): &IpAddressJob{}, (&DeviceStatusJob{}).what(): &DeviceStatusJob{}}

// StaticJob defines the interface for built-in collection jobs that generate
// data from collector state rather than protocol operations.
type StaticJob interface {
	// what returns the job name identifier
	what() string
	// do executes the static job and populates the job's Result field
	do(job *l8tpollaris.CJob, hostCollector *HostCollector)
}

// IpAddressJob is a static job that returns the IP address of the device.
// It extracts the address from the first configured protocol.
type IpAddressJob struct{}

func (this *IpAddressJob) what() string {
	return "ipAddress"
}

func (this *IpAddressJob) do(job *l8tpollaris.CJob, hostCollector *HostCollector) {
	obj := object.NewEncode()
	for _, h := range hostCollector.target.Hosts {
		for _, c := range h.Configs {
			obj.Add(c.Addr)
			job.Result = obj.Data()
			break
		}
		break
	}
}

// DeviceStatusJob is a static job that reports the online/offline status
// of each protocol collector for the device. Used during boot to report
// device reachability.
type DeviceStatusJob struct{}

func (this *DeviceStatusJob) what() string {
	return "deviceStatus"
}

func (this *DeviceStatusJob) do(job *l8tpollaris.CJob, hostCollector *HostCollector) {
	obj := object.NewEncode()
	protocolState := make(map[int32]bool)
	hostCollector.collectors.Iterate(func(k, v interface{}) {
		key := k.(l8tpollaris.L8PProtocol)
		p := v.(common.ProtocolCollector)
		protocolState[int32(key)] = p.Online()
	})
	obj.Add(protocolState)
	job.Result = obj.Data()
}

func (this *DeviceStatusJob) doDown(job *l8tpollaris.CJob, hostCollector *HostCollector) {
	obj := object.NewEncode()
	protocolState := make(map[int32]bool)
	hostCollector.collectors.Iterate(func(k, v interface{}) {
		key := k.(l8tpollaris.L8PProtocol)
		protocolState[int32(key)] = false
	})
	obj.Add(protocolState)
	job.Result = obj.Data()
}
