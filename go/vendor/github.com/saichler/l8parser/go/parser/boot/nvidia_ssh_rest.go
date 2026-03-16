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

// nvidia_ssh_rest.go provides SSH and REST polling configurations for NVIDIA GPU servers.
// These polls complement the SNMP polls in nvidia.go to achieve full GpuDevice coverage.
package boot

import (
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
)

// --- SSH polls (Polls 7-11) ---

// Poll 7: Encoder/Decoder utilization via nvidia-smi
func createNvidiaGpuUtilizationPoll(p *l8tpollaris.L8Pollaris) {
	poll := &l8tpollaris.L8Poll{}
	poll.Name = "nvidiaGpuUtilization"
	poll.What = "nvidia-smi -q -d UTILIZATION"
	poll.Protocol = l8tpollaris.L8PProtocol_L8PSSH
	poll.Cadence = EVERY_5_MINUTES_ALWAYS
	poll.Timeout = DEFAULT_TIMEOUT
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createSshNvidiaAttribute(
		"gpudevice.gpus", "utilization"))
	p.Polling[poll.Name] = poll
}

// Poll 8: Memory temp, shutdown/slowdown thresholds via nvidia-smi
func createNvidiaGpuTemperaturePoll(p *l8tpollaris.L8Pollaris) {
	poll := &l8tpollaris.L8Poll{}
	poll.Name = "nvidiaGpuTemperature"
	poll.What = "nvidia-smi -q -d TEMPERATURE"
	poll.Protocol = l8tpollaris.L8PProtocol_L8PSSH
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Timeout = DEFAULT_TIMEOUT
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createSshNvidiaAttribute(
		"gpudevice.gpus", "temperature"))
	p.Polling[poll.Name] = poll
}

// Poll 9: Power limits via nvidia-smi
func createNvidiaGpuPowerPoll(p *l8tpollaris.L8Pollaris) {
	poll := &l8tpollaris.L8Poll{}
	poll.Name = "nvidiaGpuPower"
	poll.What = "nvidia-smi -q -d POWER"
	poll.Protocol = l8tpollaris.L8PProtocol_L8PSSH
	poll.Cadence = DEFAULT_CADENCE
	poll.Timeout = DEFAULT_TIMEOUT
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createSshNvidiaAttribute(
		"gpudevice.gpus", "power"))
	p.Polling[poll.Name] = poll
}

// Poll 10: System versions, model, serial via show version
func createNvidiaVersionPoll(p *l8tpollaris.L8Pollaris) {
	poll := &l8tpollaris.L8Poll{}
	poll.Name = "nvidiaVersion"
	poll.What = "show version"
	poll.Protocol = l8tpollaris.L8PProtocol_L8PSSH
	poll.Cadence = DEFAULT_CADENCE
	poll.Timeout = DEFAULT_TIMEOUT
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createSshNvidiaAttribute(
		"gpudevice.deviceinfo", "version"))
	p.Polling[poll.Name] = poll
}

// Poll 11: CPU sockets and cores via lscpu
func createNvidiaCpuInfoPoll(p *l8tpollaris.L8Pollaris) {
	poll := &l8tpollaris.L8Poll{}
	poll.Name = "nvidiaCpuInfo"
	poll.What = "lscpu"
	poll.Protocol = l8tpollaris.L8PProtocol_L8PSSH
	poll.Cadence = DEFAULT_CADENCE
	poll.Timeout = DEFAULT_TIMEOUT
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createSshNvidiaAttribute(
		"gpudevice.system", "lscpu"))
	p.Polling[poll.Name] = poll
}

// createSshNvidiaAttribute creates an attribute that uses SshNvidiaSmiParse with a format parameter.
func createSshNvidiaAttribute(propertyId, format string) *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"gpudevice": propertyId}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	rule := &l8tpollaris.L8PRule{}
	rule.Name = "SshNvidiaSmiParse"
	rule.Params = make(map[string]*l8tpollaris.L8PParameter)
	addParameter("format", format, rule)
	attr.Rules = append(attr.Rules, rule)
	return attr
}

// --- REST polls (Polls 12-15) ---

// Poll 12: Static GPU device info (compute capability, persistence mode)
func createNvidiaGpuDevicesPoll(p *l8tpollaris.L8Pollaris) {
	poll := &l8tpollaris.L8Poll{}
	poll.Name = "nvidiaGpuDevices"
	poll.What = "GET::/api/v1/gpu/devices::"
	poll.Protocol = l8tpollaris.L8PProtocol_L8PRESTAPI
	poll.Cadence = DEFAULT_CADENCE
	poll.Timeout = DEFAULT_TIMEOUT
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createRestGpuAttribute(
		"gpudevice.gpus",
		"devices",
		"compute_capability:computecapability,persistence_mode:persistencemode"))
	p.Polling[poll.Name] = poll
}

// Poll 13: NVLink topology
func createNvidiaGpuTopologyPoll(p *l8tpollaris.L8Pollaris) {
	poll := &l8tpollaris.L8Poll{}
	poll.Name = "nvidiaGpuTopology"
	poll.What = "GET::/api/v1/gpu/topology::"
	poll.Protocol = l8tpollaris.L8PProtocol_L8PRESTAPI
	poll.Cadence = DEFAULT_CADENCE
	poll.Timeout = DEFAULT_TIMEOUT
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createRestAttribute(
		"gpudevice.topology",
		"nvlink_version:gpudevice.topology.nvlinkversion,"+
			"nvswitch_count:gpudevice.topology.nvswitchcount,"+
			"connectivity:gpudevice.topology.gpulinks"))
	p.Polling[poll.Name] = poll
}

// Poll 14: DCGM health checks
func createNvidiaDcgmHealthPoll(p *l8tpollaris.L8Pollaris) {
	poll := &l8tpollaris.L8Poll{}
	poll.Name = "nvidiaDcgmHealth"
	poll.What = "GET::/api/v1/dcgm/health::"
	poll.Protocol = l8tpollaris.L8PProtocol_L8PRESTAPI
	poll.Cadence = EVERY_5_MINUTES_ALWAYS
	poll.Timeout = DEFAULT_TIMEOUT
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createRestAttribute(
		"gpudevice.health",
		"overall_health:gpudevice.health.overallstatus,"+
			"checks:gpudevice.health.checks"))
	p.Polling[poll.Name] = poll
}

// Poll 15: System memory details
func createNvidiaSystemMemoryPoll(p *l8tpollaris.L8Pollaris) {
	poll := &l8tpollaris.L8Poll{}
	poll.Name = "nvidiaSystemMemory"
	poll.What = "GET::/api/v1/system/memory::"
	poll.Protocol = l8tpollaris.L8PProtocol_L8PRESTAPI
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Timeout = DEFAULT_TIMEOUT
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createRestAttribute(
		"gpudevice.system",
		"system_memory.free_bytes:gpudevice.system.memoryfreebytes"))
	p.Polling[poll.Name] = poll
}

// createRestGpuAttribute creates an attribute that uses RestGpuParse to extract per-GPU
// fields from a JSON array, keyed by PCI Bus ID.
func createRestGpuAttribute(propertyId, arrayPath, mapping string) *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"gpudevice": propertyId}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	rule := &l8tpollaris.L8PRule{}
	rule.Name = "RestGpuParse"
	rule.Params = make(map[string]*l8tpollaris.L8PParameter)
	addParameter("array_path", arrayPath, rule)
	addParameter("mapping", mapping, rule)
	attr.Rules = append(attr.Rules, rule)
	return attr
}

// createRestAttribute creates an attribute that uses RestJsonParse with a mapping parameter.
func createRestAttribute(propertyId, mapping string) *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"gpudevice": propertyId}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	rule := &l8tpollaris.L8PRule{}
	rule.Name = "RestJsonParse"
	rule.Params = make(map[string]*l8tpollaris.L8PParameter)
	addParameter("mapping", mapping, rule)
	attr.Rules = append(attr.Rules, rule)
	return attr
}
