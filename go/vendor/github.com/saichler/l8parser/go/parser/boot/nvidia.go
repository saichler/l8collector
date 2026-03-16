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

// nvidia.go provides polling configurations for NVIDIA GPU server devices
// (DGX A100, DGX H100, HGX H200). Populates the GpuDevice protobuf model.
package boot

import (
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
)

// NVIDIA enterprise OID prefix
const nvidiaOidBase = ".1.3.6.1.4.1.53246"
const nvidiaGpuTableOid = ".1.3.6.1.4.1.53246.1.1.1.1"
const nvidiaGpuModuleOid = ".1.3.6.1.4.1.53246.1.1.1.0"

// CreateNvidiaGpuBootPolls creates collection and parsing Pollaris model for NVIDIA GPU servers
func CreateNvidiaGpuBootPolls() *l8tpollaris.L8Pollaris {
	polaris := &l8tpollaris.L8Pollaris{}
	polaris.Name = "nvidia-gpu"
	polaris.Groups = []string{"nvidia", "nvidia-gpu"}
	polaris.Polling = make(map[string]*l8tpollaris.L8Poll)
	// SNMP polls (1-7)
	createNvidiaSystemPoll(polaris)
	createNvidiaGpuModulePoll(polaris)
	createNvidiaGpuInfoPoll(polaris)
	createNvidiaGpuMetricsPoll(polaris)
	createNvidiaHostResourcesPoll(polaris)
	createNvidiaInterfacesPoll(polaris)
	createNvidiaDeviceStatusPoll(polaris)
	// SSH polls (7-11)
	createNvidiaGpuUtilizationPoll(polaris)
	createNvidiaGpuTemperaturePoll(polaris)
	createNvidiaGpuPowerPoll(polaris)
	createNvidiaVersionPoll(polaris)
	createNvidiaCpuInfoPoll(polaris)
	// REST polls (12-15)
	createNvidiaGpuDevicesPoll(polaris)
	createNvidiaGpuTopologyPoll(polaris)
	createNvidiaDcgmHealthPoll(polaris)
	createNvidiaSystemMemoryPoll(polaris)
	// Execution order: SNMP first, then SSH, then REST
	polaris.Order = []string{
		"nvidiaSystem", "nvidiaGpuModule", "nvidiaGpuInfo", "nvidiaGpuMetrics",
		"nvidiaHostResources", "nvidiaInterfaces", "nvidiaDevStatus",
		"nvidiaGpuUtilization", "nvidiaGpuTemperature", "nvidiaGpuPower",
		"nvidiaVersion", "nvidiaCpuInfo",
		"nvidiaGpuDevices", "nvidiaGpuTopology", "nvidiaDcgmHealth", "nvidiaSystemMemory",
	}
	return polaris
}

// Poll 1: System MIB — device identification and status
func createNvidiaSystemPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("nvidiaSystem")
	poll.What = ".1.3.6.1.2.1.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createNvidiaVendor())
	poll.Attributes = append(poll.Attributes, createNvidiaHostname())
	poll.Attributes = append(poll.Attributes, createNvidiaLocation())
	poll.Attributes = append(poll.Attributes, createNvidiaUptime())
	poll.Attributes = append(poll.Attributes, createNvidiaOsVersion())
	poll.Attributes = append(poll.Attributes, createNvidiaDriverVersion())
	p.Polling[poll.Name] = poll
}

// Poll 2: NVIDIA module-level info — GPU count and DCGM version
func createNvidiaGpuModulePoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("nvidiaGpuModule")
	poll.What = nvidiaGpuModuleOid
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createNvidiaGpuCount())
	poll.Attributes = append(poll.Attributes, createNvidiaDcgmVersion())
	poll.Attributes = append(poll.Attributes, createNvidiaCudaVersion())
	p.Polling[poll.Name] = poll
}

// Poll 3: Per-GPU static info — device name, UUID, serial, PCI, driver, CUDA, ECC
func createNvidiaGpuInfoPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("nvidiaGpuInfo")
	poll.What = nvidiaGpuTableOid
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createNvidiaGpuStaticTable())
	p.Polling[poll.Name] = poll
}

// Poll 4: Per-GPU dynamic metrics — utilization, VRAM, temp, power, fan, clocks
func createNvidiaGpuMetricsPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("nvidiaGpuMetrics")
	poll.What = nvidiaGpuTableOid
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Cadence = EVERY_5_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createNvidiaGpuMetricsTable())
	p.Polling[poll.Name] = poll
}

// Poll 5: Host Resources MIB — CPU, memory, storage
func createNvidiaHostResourcesPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("nvidiaHostResources")
	poll.What = ".1.3.6.1.2.1.25"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createNvidiaMemoryTotal())
	poll.Attributes = append(poll.Attributes, createNvidiaMemoryUsed())
	poll.Attributes = append(poll.Attributes, createNvidiaCpuModel())
	poll.Attributes = append(poll.Attributes, createNvidiaCpuUtilization())
	poll.Attributes = append(poll.Attributes, createNvidiaStorageDescription())
	poll.Attributes = append(poll.Attributes, createNvidiaStorageTotal())
	poll.Attributes = append(poll.Attributes, createNvidiaStorageUsed())
	p.Polling[poll.Name] = poll
}

// Poll 6: IF-MIB — network interfaces
func createNvidiaInterfacesPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("nvidiaInterfaces")
	poll.What = ".1.3.6.1.2.1.2.2.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Table
	poll.Cadence = EVERY_15_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createNvidiaIfTable())
	p.Polling[poll.Name] = poll
}

// Poll 7: Device Status — dedicated poll for device status (same pattern as SNMP boot)
func createNvidiaDeviceStatusPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("nvidiaDevStatus")
	poll.What = "devicestatus"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Cadence = EVERY_5_MINUTES_ALWAYS
	poll.Always = true
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createNvidiaDeviceStatus())
	p.Polling[poll.Name] = poll
}

// --- System MIB attributes ---

func createNvidiaVendor() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"gpudevice": "gpudevice.deviceinfo.vendor"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createContainsRule("53246", ".1.3.6.1.2.1.1.2.0", "NVIDIA"))
	return attr
}

func createNvidiaHostname() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"gpudevice": "gpudevice.deviceinfo.hostname"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.1.5.0"))
	return attr
}

func createNvidiaLocation() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"gpudevice": "gpudevice.deviceinfo.location"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.1.6.0"))
	return attr
}

func createNvidiaUptime() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"gpudevice": "gpudevice.deviceinfo.uptime"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.1.3.0"))
	return attr
}

func createNvidiaDeviceStatus() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"gpudevice": "gpudevice.deviceinfo.devicestatus"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createDeviceStatusRule())
	return attr
}

func createNvidiaOsVersion() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"gpudevice": "gpudevice.deviceinfo.osversion"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// sysDescr typically contains OS info like "Ubuntu 22.04.3 LTS"
	attr.Rules = append(attr.Rules, createContainsRule("ubuntu", ".1.3.6.1.2.1.1.1.0", "Ubuntu"))
	return attr
}

func createNvidiaDriverVersion() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"gpudevice": "gpudevice.deviceinfo.driverversion"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// Driver version is available per-GPU but also in sysDescr
	// Use per-GPU OID for primary source (first GPU)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.53246.1.1.1.1.13.0"))
	return attr
}

// --- GPU module-level attributes ---

func createNvidiaGpuCount() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"gpudevice": "gpudevice.deviceinfo.gpucount"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.53246.1.1.1.0.1.0"))
	return attr
}

func createNvidiaDcgmVersion() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"gpudevice": "gpudevice.deviceinfo.dcgmversion"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.53246.1.1.1.0.2.0"))
	return attr
}

// --- Per-GPU table attributes (using SnmpGpuTable rule) ---

func createNvidiaGpuStaticTable() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"gpudevice": "gpudevice.gpus"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSnmpGpuTableRule(
		nvidiaGpuTableOid,
		"1:devicename:set,2:deviceuuid:set,3:serialnumber:set,4:pcibusid:set,"+
			"7:vramtotalmib:set,"+
			"13:driverversion:set,14:cudaversion:set,"+
			"15:ecccorrectedcount:set,16:eccuncorrectedcount:set",
	))
	return attr
}

func createNvidiaGpuMetricsTable() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"gpudevice": "gpudevice.gpus"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSnmpGpuTableRule(
		nvidiaGpuTableOid,
		"5:gpuutilizationpercent:ts,6:vramusedmib:ts,"+
			"8:temperaturecelsius:ts,9:powerdrawwatts:ts,10:fanspeedpercent:ts,"+
			"11:smclockmhz:ts,12:memclockmhz:ts",
	))
	return attr
}

// createSnmpGpuTableRule creates an SnmpGpuTable rule with the given OID base and mapping.
func createSnmpGpuTableRule(oidBase, mapping string) *l8tpollaris.L8PRule {
	rule := &l8tpollaris.L8PRule{}
	rule.Name = "SnmpGpuTable"
	rule.Params = make(map[string]*l8tpollaris.L8PParameter)
	addParameter("oid_base", oidBase, rule)
	addParameter("mapping", mapping, rule)
	return rule
}

// --- Host Resources MIB attributes ---

func createNvidiaMemoryTotal() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"gpudevice": "gpudevice.system.memorytotalbytes"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.25.2.2.0"))
	return attr
}

func createNvidiaCpuModel() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"gpudevice": "gpudevice.system.cpumodel"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.25.3.2.1.3.1"))
	return attr
}

func createNvidiaCpuUtilization() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"gpudevice": "gpudevice.system.cpuutilizationpercent"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.2.1.25.3.3.1.2.1"))
	return attr
}

func createNvidiaStorageDescription() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"gpudevice": "gpudevice.system.storagedescription"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.25.2.3.1.3.2"))
	return attr
}

func createNvidiaStorageTotal() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"gpudevice": "gpudevice.system.storagetotalbytes"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.25.2.3.1.5.2"))
	return attr
}

func createNvidiaStorageUsed() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"gpudevice": "gpudevice.system.storageusedbytes"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.2.1.25.2.3.1.6.2"))
	return attr
}

// --- Phase 1: Additional SNMP attributes ---

func createNvidiaCudaVersion() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"gpudevice": "gpudevice.deviceinfo.cudaversion"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// CUDA version from GPU 0 static data
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.53246.1.1.1.1.14.0"))
	return attr
}

func createNvidiaMemoryUsed() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"gpudevice": "gpudevice.system.memoryusedbytes"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// Physical Memory used from HR MIB storage index 1
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.2.1.25.2.3.1.6.1"))
	return attr
}

// --- IF-MIB interface table ---

func createNvidiaIfTable() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"gpudevice": "gpudevice.system.networkinterfaces"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// Parse IF-MIB table: 8 columns (ifIndex, ifDescr, ifType, ifSpeed, ifAdminStatus, ifOperStatus, ifInOctets, ifOutOctets)
	attr.Rules = append(attr.Rules, createToTable(8, 0))
	attr.Rules = append(attr.Rules, createTableToMap())
	return attr
}

