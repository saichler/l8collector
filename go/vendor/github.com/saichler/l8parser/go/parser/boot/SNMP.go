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

// Package boot provides SNMP and Kubernetes polling configurations for the L8Parser.
// It contains vendor-specific polling definitions for various network devices including
// Cisco, Juniper, Palo Alto, Fortinet, Arista, Nokia, Huawei, Dell, and HPE.
// The package also includes Kubernetes resource monitoring configurations.
package boot

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/saichler/l8collector/go/collector/common"
	"github.com/saichler/l8parser/go/parser/rules"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	strings2 "github.com/saichler/l8utils/go/utils/strings"
)

// DEFAULT_CADENCE defines the standard polling intervals (5min, 15min, 1hr, 2hr) for SNMP collection.
var DEFAULT_CADENCE = &l8tpollaris.L8PCadencePlan{Cadences: []int64{300, 900, 3600, 7200}, Enabled: true}

// EVERY_5_MINUTES defines polling intervals starting at 5 minutes with backoff to 1hr and 2hr.
var EVERY_5_MINUTES = &l8tpollaris.L8PCadencePlan{Cadences: []int64{300, 3600, 7200}, Enabled: true}

// EVERY_5_MINUTES_ALWAYS defines a constant 5-minute polling interval without backoff.
var EVERY_5_MINUTES_ALWAYS = &l8tpollaris.L8PCadencePlan{Cadences: []int64{300}, Enabled: true}

var EVERY_15_MINUTES_ALWAYS = &l8tpollaris.L8PCadencePlan{Cadences: []int64{900}, Enabled: true}

// DISABLED represents a disabled polling configuration with a 2-hour fallback.
var DISABLED = &l8tpollaris.L8PCadencePlan{Cadences: []int64{7200}, Enabled: false}

// DEFAULT_TIMEOUT specifies the default SNMP request timeout in seconds.
var DEFAULT_TIMEOUT int64 = 60

// stringConvert is a utility for converting values to strings with type prefixes.
var stringConvert = &strings2.String{TypesPrefix: true}

// CreateBoot00 creates the initial boot stage Pollaris model for IP address collection.
// This is the first stage in the device discovery process.
func CreateBoot00() *l8tpollaris.L8Pollaris {
	boot00 := &l8tpollaris.L8Pollaris{}
	boot00.Name = "boot00"
	boot00.Groups = []string{common.BOOT_STAGE_00}
	boot00.Polling = make(map[string]*l8tpollaris.L8Poll)
	createIpAddressPoll(boot00)
	return boot00
}

// CreateBoot01 creates the system MIB polling Pollaris model.
// This stage collects basic system information like sysName, sysOID, and vendor details.
func CreateBoot01() *l8tpollaris.L8Pollaris {
	boot01 := &l8tpollaris.L8Pollaris{}
	boot01.Name = "boot01"
	boot01.Groups = []string{common.BOOT_STAGE_01}
	boot01.Polling = make(map[string]*l8tpollaris.L8Poll)
	createSystemMibPoll(boot01)
	createInterfaceCountPoll(boot01)
	return boot01
}

// CreateBoot02 creates the device status polling Pollaris model.
// This stage monitors device availability and operational status.
func CreateBoot02() *l8tpollaris.L8Pollaris {
	boot02 := &l8tpollaris.L8Pollaris{}
	boot02.Name = "boot02"
	boot02.Groups = []string{common.BOOT_STAGE_02}
	boot02.Polling = make(map[string]*l8tpollaris.L8Poll)
	createDeviceStatusPoll(boot02)
	return boot02
}

// CreateBoot03 creates the generic SNMP collection and parsing Pollaris model.
// This stage collects interface table and entity MIB data for physical inventory.
func CreateBoot03() *l8tpollaris.L8Pollaris {
	boot03 := &l8tpollaris.L8Pollaris{}
	boot03.Name = "boot03"
	boot03.Groups = []string{common.BOOT_STAGE_03}
	boot03.Polling = make(map[string]*l8tpollaris.L8Poll)
	createIfTable(boot03)
	createEntityMibPoll(boot03)
	return boot03
}

// GetPollarisByOid returns the appropriate vendor-specific Pollaris model based on sysOID
func GetPollarisByOid(sysOid string) *l8tpollaris.L8Pollaris {
	// Cisco devices
	if isCiscoOid(sysOid) {
		if isCiscoSwitchOid(sysOid) {
			return CreateCiscoSwitchBootPolls()
		}
		return CreateCiscoRouterBootPolls() // Default to router for Cisco
	}

	// Juniper devices
	if isJuniperOid(sysOid) {
		return CreateJuniperRouterBootPolls()
	}

	// Palo Alto Networks devices
	if isPaloAltoOid(sysOid) {
		return CreatePaloAltoFirewallBootPolls()
	}

	// Fortinet devices
	if isFortinetOid(sysOid) {
		return CreateFortinetFirewallBootPolls()
	}

	// Arista devices
	if isAristaOid(sysOid) {
		return CreateAristaSwitchBootPolls()
	}

	// Nokia devices
	if isNokiaOid(sysOid) {
		return CreateNokiaRouterBootPolls()
	}

	// Huawei devices
	if isHuaweiOid(sysOid) {
		return CreateHuaweiRouterBootPolls()
	}

	// Dell servers
	if isDellOid(sysOid) {
		return CreateDellServerBootPolls()
	}

	// HPE servers
	if isHPEOid(sysOid) {
		return CreateHPEServerBootPolls()
	}

	// SonicWall devices
	if isSonicWallOid(sysOid) {
		return CreateSonicWallFirewallBootPolls()
	}

	// Check Point devices
	if isCheckPointOid(sysOid) {
		return CreateCheckPointFirewallBootPolls()
	}

	// Extreme devices
	if isExtremeOid(sysOid) {
		return CreateExtremeSwitchBootPolls()
	}

	// D-Link devices
	if isDLinkOid(sysOid) {
		return CreateDLinkSwitchBootPolls()
	}

	// IBM servers
	if isIBMOid(sysOid) {
		return CreateIBMServerBootPolls()
	}

	// NEC devices
	if isNECOid(sysOid) {
		return CreateNECRouterBootPolls()
	}

	// NVIDIA GPU servers
	if isNvidiaOid(sysOid) {
		return CreateNvidiaGpuBootPolls()
	}

	// Default to generic SNMP polling if no vendor match
	return CreateBoot03()
}

// GetAllPolarisModels returns a slice of all available Pollaris models
func GetAllPolarisModels() []*l8tpollaris.L8Pollaris {
	models := make([]*l8tpollaris.L8Pollaris, 0)

	//Generic K8s
	models = append(models, CreateK8sBootPolls())

	// Generic Pre Boot
	models = append(models, CreateBoot00())
	models = append(models, CreateBoot01())
	models = append(models, CreateBoot02())

	// Generic SNMP
	models = append(models, CreateBoot03())

	// Cisco devices
	models = append(models, CreateCiscoSwitchBootPolls())
	models = append(models, CreateCiscoRouterBootPolls())

	// Juniper devices
	models = append(models, CreateJuniperRouterBootPolls())

	// Palo Alto devices
	models = append(models, CreatePaloAltoFirewallBootPolls())

	// Fortinet devices
	models = append(models, CreateFortinetFirewallBootPolls())

	// Arista devices
	models = append(models, CreateAristaSwitchBootPolls())

	// Nokia devices
	models = append(models, CreateNokiaRouterBootPolls())

	// Huawei devices
	models = append(models, CreateHuaweiRouterBootPolls())

	// Dell devices
	models = append(models, CreateDellServerBootPolls())

	// HPE devices
	models = append(models, CreateHPEServerBootPolls())

	// SonicWall devices
	models = append(models, CreateSonicWallFirewallBootPolls())

	// Check Point devices
	models = append(models, CreateCheckPointFirewallBootPolls())

	// Extreme devices
	models = append(models, CreateExtremeSwitchBootPolls())

	// D-Link devices
	models = append(models, CreateDLinkSwitchBootPolls())

	// IBM devices
	models = append(models, CreateIBMServerBootPolls())

	// NEC devices
	models = append(models, CreateNECRouterBootPolls())

	// NVIDIA GPU servers
	models = append(models, CreateNvidiaGpuBootPolls())

	return models
}

// OID matching helper functions
func isCiscoOid(sysOid string) bool {
	// Normalize OID by ensuring it starts with a dot
	normalizedOid := sysOid
	if !strings.HasPrefix(normalizedOid, ".") {
		normalizedOid = "." + normalizedOid
	}
	return strings.HasPrefix(normalizedOid, ".1.3.6.1.4.1.9.")
}

func isCiscoSwitchOid(sysOid string) bool {
	// Normalize OID by ensuring it starts with a dot
	normalizedOid := sysOid
	if !strings.HasPrefix(normalizedOid, ".") {
		normalizedOid = "." + normalizedOid
	}

	// Cisco switch specific OIDs (Catalyst series)
	switchOids := []string{
		".1.3.6.1.4.1.9.1.122",  // Catalyst 2960
		".1.3.6.1.4.1.9.1.616",  // Catalyst 3560
		".1.3.6.1.4.1.9.1.717",  // Catalyst 3750
		".1.3.6.1.4.1.9.1.1208", // Catalyst 4500
		".1.3.6.1.4.1.9.1.1146", // Catalyst 6500
	}
	for _, switchOid := range switchOids {
		if strings.HasPrefix(normalizedOid, switchOid) {
			return true
		}
	}
	return false
}

func isJuniperOid(sysOid string) bool {
	// Normalize OID by ensuring it starts with a dot
	normalizedOid := sysOid
	if !strings.HasPrefix(normalizedOid, ".") {
		normalizedOid = "." + normalizedOid
	}
	return strings.HasPrefix(normalizedOid, ".1.3.6.1.4.1.2636.")
}

func isPaloAltoOid(sysOid string) bool {
	// Normalize OID by ensuring it starts with a dot
	normalizedOid := sysOid
	if !strings.HasPrefix(normalizedOid, ".") {
		normalizedOid = "." + normalizedOid
	}
	return strings.HasPrefix(normalizedOid, ".1.3.6.1.4.1.25461.")
}

func isFortinetOid(sysOid string) bool {
	// Normalize OID by ensuring it starts with a dot
	normalizedOid := sysOid
	if !strings.HasPrefix(normalizedOid, ".") {
		normalizedOid = "." + normalizedOid
	}
	return strings.HasPrefix(normalizedOid, ".1.3.6.1.4.1.12356.")
}

func isAristaOid(sysOid string) bool {
	// Normalize OID by ensuring it starts with a dot
	normalizedOid := sysOid
	if !strings.HasPrefix(normalizedOid, ".") {
		normalizedOid = "." + normalizedOid
	}
	return strings.HasPrefix(normalizedOid, ".1.3.6.1.4.1.30065.")
}

func isNokiaOid(sysOid string) bool {
	// Normalize OID by ensuring it starts with a dot
	normalizedOid := sysOid
	if !strings.HasPrefix(normalizedOid, ".") {
		normalizedOid = "." + normalizedOid
	}
	return strings.HasPrefix(normalizedOid, ".1.3.6.1.4.1.6527.")
}

func isHuaweiOid(sysOid string) bool {
	// Normalize OID by ensuring it starts with a dot
	normalizedOid := sysOid
	if !strings.HasPrefix(normalizedOid, ".") {
		normalizedOid = "." + normalizedOid
	}
	return strings.HasPrefix(normalizedOid, ".1.3.6.1.4.1.2011.")
}

func isDellOid(sysOid string) bool {
	// Normalize OID by ensuring it starts with a dot
	normalizedOid := sysOid
	if !strings.HasPrefix(normalizedOid, ".") {
		normalizedOid = "." + normalizedOid
	}
	return strings.HasPrefix(normalizedOid, ".1.3.6.1.4.1.674.")
}

func isHPEOid(sysOid string) bool {
	// Normalize OID by ensuring it starts with a dot
	normalizedOid := sysOid
	if !strings.HasPrefix(normalizedOid, ".") {
		normalizedOid = "." + normalizedOid
	}
	return strings.HasPrefix(normalizedOid, ".1.3.6.1.4.1.232.")
}

func isSonicWallOid(sysOid string) bool {
	normalizedOid := sysOid
	if !strings.HasPrefix(normalizedOid, ".") {
		normalizedOid = "." + normalizedOid
	}
	return strings.HasPrefix(normalizedOid, ".1.3.6.1.4.1.8714.")
}

func isCheckPointOid(sysOid string) bool {
	normalizedOid := sysOid
	if !strings.HasPrefix(normalizedOid, ".") {
		normalizedOid = "." + normalizedOid
	}
	return strings.HasPrefix(normalizedOid, ".1.3.6.1.4.1.2620.")
}

func isExtremeOid(sysOid string) bool {
	normalizedOid := sysOid
	if !strings.HasPrefix(normalizedOid, ".") {
		normalizedOid = "." + normalizedOid
	}
	return strings.HasPrefix(normalizedOid, ".1.3.6.1.4.1.1916.")
}

func isDLinkOid(sysOid string) bool {
	normalizedOid := sysOid
	if !strings.HasPrefix(normalizedOid, ".") {
		normalizedOid = "." + normalizedOid
	}
	return strings.HasPrefix(normalizedOid, ".1.3.6.1.4.1.171.")
}

func isIBMOid(sysOid string) bool {
	normalizedOid := sysOid
	if !strings.HasPrefix(normalizedOid, ".") {
		normalizedOid = "." + normalizedOid
	}
	return strings.HasPrefix(normalizedOid, ".1.3.6.1.4.1.2.6.")
}

func isNECOid(sysOid string) bool {
	normalizedOid := sysOid
	if !strings.HasPrefix(normalizedOid, ".") {
		normalizedOid = "." + normalizedOid
	}
	return strings.HasPrefix(normalizedOid, ".1.3.6.1.4.1.119.")
}

func isNvidiaOid(sysOid string) bool {
	normalizedOid := sysOid
	if !strings.HasPrefix(normalizedOid, ".") {
		normalizedOid = "." + normalizedOid
	}
	return strings.HasPrefix(normalizedOid, ".1.3.6.1.4.1.53246.")
}

func createSystemMibPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("systemMib")
	poll.What = ".1.3.6.1.2.1.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Cadence = EVERY_5_MINUTES
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createVendor())            // networkdevice.equipmentinfo.vendor
	poll.Attributes = append(poll.Attributes, createSysName())           // networkdevice.equipmentinfo.sys_name
	poll.Attributes = append(poll.Attributes, createSysOid())            // networkdevice.equipmentinfo.sys_oid
	poll.Attributes = append(poll.Attributes, createSystemDescription()) // networkdevice.equipmentinfo.hardware
	poll.Attributes = append(poll.Attributes, createSystemSoftware())    // networkdevice.equipmentinfo.software
	poll.Attributes = append(poll.Attributes, createSystemVersion())     // networkdevice.equipmentinfo.version
	poll.Attributes = append(poll.Attributes, createSystemModel())       // networkdevice.equipmentinfo.model
	poll.Attributes = append(poll.Attributes, createSystemUptime())      // networkdevice.equipmentinfo.uptime
	poll.Attributes = append(poll.Attributes, createSystemLocation())    // networkdevice.equipmentinfo.location
	poll.Attributes = append(poll.Attributes, createSystemDeviceType())  // networkdevice.equipmentinfo.device_type
	p.Polling[poll.Name] = poll
}

func createInterfaceCountPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("interfaceCount")
	poll.What = ".1.3.6.1.2.1.2.1.0"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Get
	poll.Cadence = EVERY_5_MINUTES
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createInterfaceCountAttribute())
	p.Polling[poll.Name] = poll
}

func createIfTable(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("ifTable")
	poll.What = ".1.3.6.1.2.1.2.2"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Table
	poll.Cadence = DISABLED // Disable ifTable polling
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createIfTableRule())
	p.Polling[poll.Name] = poll
}

func createEntityMibPoll(p *l8tpollaris.L8Pollaris) {
	// Table poll for EntityMibToPhysicals custom rule (needs CTable input)
	poll := createBaseSNMPPoll("entityMib")
	poll.What = ".1.3.6.1.2.1.47.1.1"
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Table
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createEntityMibRule())
	p.Polling[poll.Name] = poll

	// Map poll for standard Entity MIB attributes using Set rules (needs CMap input)
	mapPoll := createBaseSNMPPoll("entityMibAttributes")
	mapPoll.What = ".1.3.6.1.2.1.47.1.1.1.1"
	mapPoll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	mapPoll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)

	// Standard Entity MIB chassis attributes (RFC 2737)
	mapPoll.Attributes = append(mapPoll.Attributes, createChassisSerialAttribute())
	mapPoll.Attributes = append(mapPoll.Attributes, createChassisModelAttribute())
	mapPoll.Attributes = append(mapPoll.Attributes, createChassisDescriptionAttribute())
	mapPoll.Attributes = append(mapPoll.Attributes, createChassisComponentStatus())

	// Standard Entity MIB module attributes
	mapPoll.Attributes = append(mapPoll.Attributes, createModuleName())
	mapPoll.Attributes = append(mapPoll.Attributes, createModuleModel())
	mapPoll.Attributes = append(mapPoll.Attributes, createModuleStatus())
	mapPoll.Attributes = append(mapPoll.Attributes, createModuleNameAttribute())
	mapPoll.Attributes = append(mapPoll.Attributes, createModuleDescriptionAttribute())
	mapPoll.Attributes = append(mapPoll.Attributes, createModuleTypeAttribute())
	mapPoll.Attributes = append(mapPoll.Attributes, createRouteProcessorStatus())
	mapPoll.Attributes = append(mapPoll.Attributes, createCardStatus())

	// Standard Entity MIB CPU attributes
	mapPoll.Attributes = append(mapPoll.Attributes, createCpuIdAttribute())
	mapPoll.Attributes = append(mapPoll.Attributes, createCpuNameAttribute())
	mapPoll.Attributes = append(mapPoll.Attributes, createCpuModelAttribute())
	mapPoll.Attributes = append(mapPoll.Attributes, createCpuArchitectureAttribute())
	mapPoll.Attributes = append(mapPoll.Attributes, createCpuStatusAttribute())

	// Standard Entity MIB memory attributes
	mapPoll.Attributes = append(mapPoll.Attributes, createMemoryIdAttribute())
	mapPoll.Attributes = append(mapPoll.Attributes, createMemoryNameAttribute())
	mapPoll.Attributes = append(mapPoll.Attributes, createMemoryTypeAttribute())
	mapPoll.Attributes = append(mapPoll.Attributes, createMemoryStatusAttribute())

	// Standard Entity MIB power supply attributes
	mapPoll.Attributes = append(mapPoll.Attributes, createPowerSupplyIdAttribute())
	mapPoll.Attributes = append(mapPoll.Attributes, createPowerSupplyNameAttribute())
	mapPoll.Attributes = append(mapPoll.Attributes, createPowerSupplyModelAttribute())
	mapPoll.Attributes = append(mapPoll.Attributes, createPowerSupplySerialNumberAttribute())
	mapPoll.Attributes = append(mapPoll.Attributes, createPowerSupplyStatusAttribute())
	mapPoll.Attributes = append(mapPoll.Attributes, createPowerSupplyPowerTypeAttribute())
	mapPoll.Attributes = append(mapPoll.Attributes, createPowerSupplyStatus())
	mapPoll.Attributes = append(mapPoll.Attributes, createPowerSupplyModel())

	// Standard Entity MIB fan attributes
	mapPoll.Attributes = append(mapPoll.Attributes, createFanIdAttribute())
	mapPoll.Attributes = append(mapPoll.Attributes, createFanNameAttribute())
	mapPoll.Attributes = append(mapPoll.Attributes, createFanDescriptionAttribute())
	mapPoll.Attributes = append(mapPoll.Attributes, createFanStatusAttribute())
	mapPoll.Attributes = append(mapPoll.Attributes, createFanStatus())

	// Standard Entity MIB temperature sensors
	mapPoll.Attributes = append(mapPoll.Attributes, createTemperatureSensors())

	// EquipmentInfo attributes from Entity MIB (RFC 4133)
	mapPoll.Attributes = append(mapPoll.Attributes, createFirmwareVersionAttribute())
	mapPoll.Attributes = append(mapPoll.Attributes, createVendorTypeOidAttribute())
	mapPoll.Attributes = append(mapPoll.Attributes, createPhysicalAliasAttribute())
	mapPoll.Attributes = append(mapPoll.Attributes, createAssetIdAttribute())
	mapPoll.Attributes = append(mapPoll.Attributes, createIsFruAttribute())
	mapPoll.Attributes = append(mapPoll.Attributes, createManufacturingDateAttribute())
	mapPoll.Attributes = append(mapPoll.Attributes, createManufacturerNameAttribute())
	mapPoll.Attributes = append(mapPoll.Attributes, createIdentificationUrisAttribute())

	p.Polling[mapPoll.Name] = mapPoll
}

func createIpAddressPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("ipAddress")
	poll.What = "ipaddress" // Static value instead of SNMP OID
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Cadence = EVERY_5_MINUTES
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createIpAddress())
	p.Polling[poll.Name] = poll
}

func createDeviceStatusPoll(p *l8tpollaris.L8Pollaris) {
	poll := createBaseSNMPPoll("deviceStatus")
	poll.What = "devicestatus" // Static value instead of SNMP OID
	poll.Operation = l8tpollaris.L8C_Operation_L8C_Map
	poll.Cadence = EVERY_5_MINUTES_ALWAYS
	poll.Attributes = make([]*l8tpollaris.L8PAttribute, 0)
	poll.Attributes = append(poll.Attributes, createDeviceStatus())
	poll.Always = true
	p.Polling[poll.Name] = poll
}

func createVendor() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.vendor", "gpudevice": "gpudevice.deviceinfo.vendor"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createContainsRule("cisco", ".1.3.6.1.2.1.1.1.0", "Cisco"))
	attr.Rules = append(attr.Rules, createContainsRule("ubuntu", ".1.3.6.1.2.1.1.1.0", "Ubuntu Linux"))
	return attr
}

func createIfTableRule() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)

	// Use custom rule to translate ifTable CTable to NetworkDevice.physicals
	rule := &l8tpollaris.L8PRule{}
	rule.Name = "IfTableToPhysicals"
	rule.Params = make(map[string]*l8tpollaris.L8PParameter)
	attr.Rules = append(attr.Rules, rule)

	return attr
}

func createEntityMibRule() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)

	// Use custom rule to translate Entity MIB CTable to NetworkDevice.physicals
	rule := &l8tpollaris.L8PRule{}
	rule.Name = "EntityMibToPhysicals"
	rule.Params = make(map[string]*l8tpollaris.L8PParameter)
	attr.Rules = append(attr.Rules, rule)

	return attr
}

func createSysOid() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.sysoid"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.1.2.0"))
	return attr
}

func createSysName() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.sysname", "gpudevice": "gpudevice.deviceinfo.hostname"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.1.5.0"))
	return attr
}

func createVersion() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.vendor", "gpudevice": "gpudevice.deviceinfo.vendor"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createContainsRule("cisco", ".1.3.6.1.2.1.1.1.0", "Cisco"))
	attr.Rules = append(attr.Rules, createContainsRule("ubuntu", ".1.3.6.1.2.1.1.1.0", "Ubuntu Linux"))
	return attr
}

// Common utility functions for creating rules and polls
func addParameter(name, value string, rule *l8tpollaris.L8PRule) {
	param := &l8tpollaris.L8PParameter{}
	param.Name = name
	param.Value = value
	rule.Params[name] = param
}

func createContainsRule(what, from, output string) *l8tpollaris.L8PRule {
	rule := &l8tpollaris.L8PRule{}
	rule.Name = "Contains"
	rule.Params = make(map[string]*l8tpollaris.L8PParameter)
	addParameter("what", what, rule)
	addParameter("from", from, rule)
	addParameter("output", output, rule)
	return rule
}

func createToTable(columns int, keycolumn ...int) *l8tpollaris.L8PRule {
	rule := &l8tpollaris.L8PRule{}
	rule.Name = "StringToCTable"
	rule.Params = make(map[string]*l8tpollaris.L8PParameter)
	rule.Params[rules.Columns] = &l8tpollaris.L8PParameter{Name: rules.Columns, Value: strconv.Itoa(columns)}
	keyStr := stringConvert.ToString(reflect.ValueOf(keycolumn))
	rule.Params[rules.KeyColumn] = &l8tpollaris.L8PParameter{Name: rules.KeyColumn, Value: keyStr}
	return rule
}

func createTableToMap() *l8tpollaris.L8PRule {
	rule := &l8tpollaris.L8PRule{}
	rule.Name = "CTableToMapProperty"
	rule.Params = make(map[string]*l8tpollaris.L8PParameter)
	return rule
}

func createSetRule(from string) *l8tpollaris.L8PRule {
	rule := &l8tpollaris.L8PRule{}
	rule.Name = "Set"
	rule.Params = make(map[string]*l8tpollaris.L8PParameter)
	addParameter("from", from, rule)
	return rule
}

func createSetTimeSeriesRule(from string) *l8tpollaris.L8PRule {
	rule := &l8tpollaris.L8PRule{}
	rule.Name = "SetTimeSeries"
	rule.Params = make(map[string]*l8tpollaris.L8PParameter)
	addParameter("from", from, rule)
	return rule
}

func createDeviceStatusRule() *l8tpollaris.L8PRule {
	rule := &l8tpollaris.L8PRule{}
	rule.Name = "MapToDeviceStatus"
	rule.Params = make(map[string]*l8tpollaris.L8PParameter)
	addParameter("from", "devicestatus", rule)
	return rule
}

// createNormalizeEnumRule creates a NormalizeEnum rule with the given value mapping.
// The mapping format is "inputVal:outputVal,inputVal:outputVal,...,*:defaultVal".
// The special key "*" provides a fallback for any unmapped values (defaults to 0).
func createNormalizeEnumRule(mapping string) *l8tpollaris.L8PRule {
	rule := &l8tpollaris.L8PRule{}
	rule.Name = "NormalizeEnum"
	rule.Params = make(map[string]*l8tpollaris.L8PParameter)
	addParameter("map", mapping, rule)
	return rule
}

// ComponentStatus enum values:
//
//	0 = UNKNOWN, 1 = OK, 2 = WARNING, 3 = ERROR, 4 = CRITICAL, 5 = OFFLINE, 6 = NOT_PRESENT
//
// entPhysicalClass (OID .5) values:
//
//	1=other, 2=unknown, 3=chassis, 4=backplane, 5=container, 6=PSU, 7=fan,
//	8=sensor, 9=module, 10=port, 11=stack, 12=cpu
//
// Since entPhysicalClass reports what TYPE the component is (not its operational status),
// a valid response for any class means the component is present and OK.
const normalizeEntPhysClassToComponentStatus = "1:1,2:0,3:1,4:1,5:1,6:1,7:1,8:1,9:1,10:1,11:1,12:1,*:0"

// ModuleType enum values:
//
//	0=UNKNOWN, 1=SUPERVISOR, 2=LINE_CARD, 3=ROUTE_PROCESSOR, 4=INTERFACE_MODULE,
//	5=MANAGEMENT_PROCESSOR, 6=SECURITY_PROCESSING_UNIT, 7=SERVICE_MODULE, 8=FABRIC_MODULE
//
// Mapping from entPhysicalClass to ModuleType:
const normalizeEntPhysClassToModuleType = "9:2,12:3,4:8,*:0"

func createBaseSNMPPoll(jobName string) *l8tpollaris.L8Poll {
	poll := &l8tpollaris.L8Poll{}
	poll.Name = jobName
	poll.Timeout = DEFAULT_TIMEOUT
	poll.Cadence = DEFAULT_CADENCE
	poll.Protocol = l8tpollaris.L8PProtocol_L8PPSNMPV2
	return poll
}

// Common network attribute functions (used by vendor-specific files)
func createInterfaceName() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.ports.interfaces.name"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.2.2.1.2.1"))
	return attr
}

func createInterfaceStatus() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.ports.interfaces.status"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.2.2.1.8.1"))
	return attr
}

func createInterfaceSpeed() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.ports.interfaces.speed"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.2.2.1.5.1"))
	return attr
}

func createInterfaceMtu() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.ports.interfaces.mtu"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.2.2.1.4.1"))
	return attr
}

// Module and chassis attribute functions
func createModuleName() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.modules.name"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.2.1"))
	return attr
}

func createModuleModel() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.modules.model"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.13.1"))
	return attr
}

func createModuleStatus() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.modules.status"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.5.1"))
	attr.Rules = append(attr.Rules, createNormalizeEnumRule(normalizeEntPhysClassToComponentStatus))
	return attr
}

func createChassisComponentStatus() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.status"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.6.1"))
	attr.Rules = append(attr.Rules, createNormalizeEnumRule(normalizeEntPhysClassToComponentStatus))
	return attr
}

// Power and environmental attribute functions
func createPowerSupplyStatus() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.powersupplies.status"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.3.1"))
	attr.Rules = append(attr.Rules, createNormalizeEnumRule(normalizeEntPhysClassToComponentStatus))
	return attr
}

func createPowerSupplyModel() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.powersupplies.model"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.2.1"))
	return attr
}

func createFanStatus() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.fans.status"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.3.1"))
	attr.Rules = append(attr.Rules, createNormalizeEnumRule(normalizeEntPhysClassToComponentStatus))
	return attr
}

func createTemperatureSensors() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.performance.temperaturecelsius"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.4.1"))
	return attr
}

// Performance attribute functions
func createCpuUtilization() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.performance.cpuusagepercent"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.9.9.109.1.1.1.1.5.1"))
	return attr
}

func createMemoryUtilization() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.performance.memoryusagepercent"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetTimeSeriesRule(".1.3.6.1.4.1.9.9.48.1.1.1.6.1"))
	return attr
}

// Router-specific attribute functions
func createRouteProcessorStatus() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.modules.status"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.5.1"))
	attr.Rules = append(attr.Rules, createNormalizeEnumRule(normalizeEntPhysClassToComponentStatus))
	return attr
}

func createRoutingTableEntry() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.logicals.interfaces.ipaddress"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.4.21.1.1.1"))
	return attr
}

func createCardStatus() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.modules.status"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.9.1"))
	attr.Rules = append(attr.Rules, createNormalizeEnumRule(normalizeEntPhysClassToComponentStatus))
	return attr
}

// Firewall-specific attribute functions
func createActiveSessions() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.performance.activeconnections"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.25461.2.1.2.3.1.0"))
	return attr
}

func createThreatCount() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.networkhealth.alertcount"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.25461.2.1.2.2.1.0"))
	return attr
}

func createVpnTunnelStatus() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.networklinks.linkstatus"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.4.1.25461.2.1.2.4.1.3.1"))
	return attr
}

// Server-specific attribute functions
func createDiskStatus() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.performance.processes.status"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.25.2.3.1.4.1"))
	return attr
}

// =====================================
// COMPREHENSIVE POLLING CONFIGURATIONS
// Supporting all NetworkDevice model attributes
// =====================================

// Equipment Info Extended Attributes
func createSysNameAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.sysname", "gpudevice": "gpudevice.deviceinfo.hostname"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.1.5.0"))
	return attr
}

func createSysOidAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.sysoid"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.1.2.0"))
	return attr
}

func createTargetIdAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.deviceid"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// NOTE: TargetId typically derived from inventory system, not directly available via SNMP
	// Could be mapped from sysName or other identifier
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.1.5.0")) // Use sysName as fallback
	return attr
}

func createLocationAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.location", "gpudevice": "gpudevice.deviceinfo.location"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.1.6.0")) // sysLocation
	return attr
}

func createUptimeAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.uptime", "gpudevice": "gpudevice.deviceinfo.uptime"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.1.3.0")) // sysUpTime
	return attr
}

func createLastSeenAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.lastseen", "gpudevice": "gpudevice.deviceinfo.lastseen"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// NOTE: LastSeen is typically managed by polling system, not available via SNMP
	// This would be updated by the collector based on successful polling
	return attr
}

func createHardwareAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.hardware"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.1.1.0")) // sysDescr (contains hardware info)
	return attr
}

func createSoftwareAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.software", "gpudevice": "gpudevice.deviceinfo.osversion"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.1.1.0")) // sysDescr (contains software info)
	return attr
}

func createSeriesAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.series"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// NOTE: Series typically derived from model parsing, not directly available
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.1.1.0")) // Parse from sysDescr
	return attr
}

func createFamilyAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.family"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// NOTE: Family typically derived from model parsing, not directly available
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.1.1.0")) // Parse from sysDescr
	return attr
}

// Physical Component Attributes
func createChassisSerialAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.serialnumber"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.11.1")) // entPhysicalSerialNum
	return attr
}

func createChassisModelAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.model"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.13.1")) // entPhysicalModelName
	return attr
}

func createChassisDescriptionAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.description"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.2.1")) // entPhysicalDescr
	return attr
}

func createChassisTemperatureAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.temperature"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.99.1.1.1.4.1")) // entSensorValue (temperature)
	return attr
}

// Module Attributes
func createModuleNameAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.modules.name"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.7.1")) // entPhysicalName (table)
	return attr
}

func createModuleModelAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.modules.model"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.13.1")) // entPhysicalModelName (table)
	return attr
}

func createModuleDescriptionAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.modules.description"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.2.1")) // entPhysicalDescr (table)
	return attr
}

func createModuleStatusAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.modules.status"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.5.1")) // entPhysicalClass (table)
	attr.Rules = append(attr.Rules, createNormalizeEnumRule(normalizeEntPhysClassToComponentStatus))
	return attr
}

func createModuleTypeAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.modules.moduletype"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// entPhysicalClass: 9=module→LINE_CARD, 12=cpu→ROUTE_PROCESSOR, 4=backplane→FABRIC_MODULE
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.5.1"))
	attr.Rules = append(attr.Rules, createNormalizeEnumRule(normalizeEntPhysClassToModuleType))
	return attr
}

func createModuleTemperatureAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.modules.temperature"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.99.1.1.1.4.1")) // entSensorValue (table)
	return attr
}

// CPU Attributes
func createCpuIdAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.modules.cpus.id"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.1.1")) // entPhysicalIndex for CPU
	return attr
}

func createCpuNameAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.modules.cpus.name"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.7.1")) // entPhysicalName for CPU
	return attr
}

func createCpuModelAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.modules.cpus.model"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.13.1")) // entPhysicalModelName for CPU
	return attr
}

func createCpuArchitectureAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.modules.cpus.architecture"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// NOTE: Architecture not typically available via standard SNMP, would need vendor-specific MIBs
	// Placeholder using description field
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.2.1"))
	return attr
}

func createCpuCoresAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.modules.cpus.cores"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// NOTE: Core count not typically available via standard SNMP
	// Would require vendor-specific MIBs or parsing from description
	return attr
}

func createCpuFrequencyAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.modules.cpus.frequencymhz"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// NOTE: CPU frequency not typically available via standard SNMP
	// Would require vendor-specific MIBs
	return attr
}

func createCpuStatusAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.modules.cpus.status"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.5.1")) // entPhysicalClass
	attr.Rules = append(attr.Rules, createNormalizeEnumRule(normalizeEntPhysClassToComponentStatus))
	return attr
}

func createCpuTemperatureAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.modules.cpus.temperature"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.99.1.1.1.4.1")) // entSensorValue for CPU temp
	return attr
}

// Memory Attributes
func createMemoryIdAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.modules.memorymodules.id"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.1.1")) // entPhysicalIndex for Memory
	return attr
}

func createMemoryNameAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.modules.memorymodules.name"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.7.1")) // entPhysicalName for Memory
	return attr
}

func createMemoryTypeAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.modules.memorymodules.type"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// NOTE: Memory type not typically available via standard SNMP
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.2.1")) // Parse from description
	return attr
}

func createMemorySizeAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.modules.memorymodules.sizebytes"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// NOTE: Memory size not typically available via standard SNMP for modules
	// Would use HOST-RESOURCES-MIB for total memory
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.25.2.2.0")) // hrMemorySize
	return attr
}

func createMemoryFrequencyAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.modules.memorymodules.frequencymhz"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// NOTE: Memory frequency not typically available via standard SNMP
	return attr
}

func createMemoryStatusAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.chassis.modules.memorymodules.status"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.5.1")) // entPhysicalClass
	attr.Rules = append(attr.Rules, createNormalizeEnumRule(normalizeEntPhysClassToComponentStatus))
	return attr
}

// Port Attributes
func createPortIdAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.ports.id"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.2.2.1.1.1")) // ifIndex (table)
	return attr
}

// Interface Attributes (nested in ports)
func createInterfaceIdAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.ports.interfaces.id"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.2.2.1.1.1")) // ifIndex
	return attr
}

func createInterfaceNameAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.ports.interfaces.name"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.2.2.1.2.1")) // ifDescr
	return attr
}

func createInterfaceStatusAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.ports.interfaces.status"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.2.2.1.8.1")) // ifOperStatus
	return attr
}

func createInterfaceDescriptionAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.ports.interfaces.description"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.31.1.1.1.18.1")) // ifAlias
	return attr
}

func createInterfaceTypeAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.ports.interfaces.interfacetype"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.2.2.1.3.1")) // ifType
	return attr
}

func createInterfaceSpeedAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.ports.interfaces.speed"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.2.2.1.5.1")) // ifSpeed
	return attr
}

func createInterfaceMacAddressAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.ports.interfaces.macaddress"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.2.2.1.6.1")) // ifPhysAddress
	return attr
}

func createInterfaceIpAddressAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.ports.interfaces.ipaddress"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.4.20.1.1.1")) // ipAdEntAddr (table)
	return attr
}

func createInterfaceMtuAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.ports.interfaces.mtu"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.2.2.1.4.1")) // ifMtu
	return attr
}

func createInterfaceAdminStatusAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.ports.interfaces.adminstatus"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.2.2.1.7.1")) // ifAdminStatus
	return attr
}

// Interface Statistics
func createInterfaceRxPacketsAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.ports.interfaces.statistics.rxpackets"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.2.2.1.11.1")) // ifInUcastPkts
	return attr
}

func createInterfaceTxPacketsAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.ports.interfaces.statistics.txpackets"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.2.2.1.17.1")) // ifOutUcastPkts
	return attr
}

func createInterfaceRxBytesAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.ports.interfaces.statistics.rxbytes"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.2.2.1.10.1")) // ifInOctets
	return attr
}

func createInterfaceTxBytesAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.ports.interfaces.statistics.txbytes"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.2.2.1.16.1")) // ifOutOctets
	return attr
}

func createInterfaceRxErrorsAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.ports.interfaces.statistics.rxerrors"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.2.2.1.14.1")) // ifInErrors
	return attr
}

func createInterfaceTxErrorsAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.ports.interfaces.statistics.txerrors"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.2.2.1.20.1")) // ifOutErrors
	return attr
}

func createInterfaceRxDropsAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.ports.interfaces.statistics.rxdrops"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.2.2.1.13.1")) // ifInDiscards
	return attr
}

func createInterfaceTxDropsAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.ports.interfaces.statistics.txdrops"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.2.2.1.19.1")) // ifOutDiscards
	return attr
}

func createInterfaceCollisionsAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.ports.interfaces.statistics.collisions"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// NOTE: Collisions not available in standard IF-MIB, would need EtherLike-MIB
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.10.7.2.1.4.1")) // dot3StatsLateCollisions
	return attr
}

// Power Supply Attributes
func createPowerSupplyIdAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.powersupplies.id"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.1.1")) // entPhysicalIndex for PSU
	return attr
}

func createPowerSupplyNameAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.powersupplies.name"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.7.1")) // entPhysicalName
	return attr
}

func createPowerSupplyModelAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.powersupplies.model"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.13.1")) // entPhysicalModelName
	return attr
}

func createPowerSupplySerialNumberAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.powersupplies.serialnumber"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.11.1")) // entPhysicalSerialNum
	return attr
}

func createPowerSupplyWattageAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.powersupplies.wattage"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// NOTE: Wattage not typically available via standard SNMP, vendor-specific required
	return attr
}

func createPowerSupplyPowerTypeAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.powersupplies.powertype"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// NOTE: Power type (AC/DC) not available via standard SNMP Entity MIB.
	// entPhysicalDescr returns a description string (e.g., "IBM Power System S922 Server")
	// which cannot be mapped to the PowerType enum. Vendor-specific MIBs are needed.
	return attr
}

func createPowerSupplyStatusAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.powersupplies.status"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.5.1")) // entPhysicalClass
	attr.Rules = append(attr.Rules, createNormalizeEnumRule(normalizeEntPhysClassToComponentStatus))
	return attr
}

func createPowerSupplyTemperatureAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.powersupplies.temperature"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.99.1.1.1.4.1")) // entSensorValue for PSU temp
	return attr
}

func createPowerSupplyLoadPercentAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.powersupplies.loadpercent"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// NOTE: Load percentage not typically available via standard SNMP
	return attr
}

func createPowerSupplyVoltageAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.powersupplies.voltage"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// NOTE: Voltage not typically available via standard SNMP
	return attr
}

func createPowerSupplyCurrentAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.powersupplies.current"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// NOTE: Current not typically available via standard SNMP
	return attr
}

// Fan Attributes
func createFanIdAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.fans.id"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.1.1")) // entPhysicalIndex for Fan
	return attr
}

func createFanNameAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.fans.name"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.7.1")) // entPhysicalName
	return attr
}

func createFanDescriptionAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.fans.description"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.2.1")) // entPhysicalDescr
	return attr
}

func createFanStatusAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.fans.status"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.5.1")) // entPhysicalClass
	attr.Rules = append(attr.Rules, createNormalizeEnumRule(normalizeEntPhysClassToComponentStatus))
	return attr
}

func createFanSpeedRpmAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.fans.speedrpm"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.99.1.1.1.4.1")) // entSensorValue for fan speed
	return attr
}

func createFanMaxSpeedRpmAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.fans.maxspeedrpm"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// NOTE: Max speed not typically available via standard SNMP
	return attr
}

func createFanTemperatureAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.fans.temperature"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.99.1.1.1.4.1")) // entSensorValue for fan temp
	return attr
}

func createFanVariableSpeedAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.physicals.fans.variablespeed"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// NOTE: Variable speed capability not typically available via standard SNMP
	return attr
}

// =====================================
// LOGICAL INTERFACES SECTION
// =====================================

// Logical Interface Attributes
func createLogicalInterfaceIdAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.logicals.interfaces.id"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.2.2.1.1.1")) // ifIndex for logical interfaces
	return attr
}

func createLogicalInterfaceNameAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.logicals.interfaces.name"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.2.2.1.2.1")) // ifDescr for logical interfaces
	return attr
}

func createLogicalInterfaceStatusAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.logicals.interfaces.status"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.2.2.1.8.1")) // ifOperStatus
	return attr
}

func createLogicalInterfaceDescriptionAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.logicals.interfaces.description"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.31.1.1.1.18.1")) // ifAlias
	return attr
}

func createLogicalInterfaceTypeAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.logicals.interfaces.interfacetype"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.2.2.1.3.1")) // ifType
	return attr
}

func createLogicalInterfaceIpAddressAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.logicals.interfaces.ipaddress"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.4.20.1.1.1")) // ipAdEntAddr (table)
	return attr
}

func createLogicalInterfaceMtuAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.logicals.interfaces.mtu"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.2.2.1.4.1")) // ifMtu
	return attr
}

func createLogicalInterfaceAdminStatusAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.logicals.interfaces.adminstatus"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.2.2.1.7.1")) // ifAdminStatus
	return attr
}

// =====================================
// NETWORK TOPOLOGY SECTION
// =====================================

// Network Topology Attributes
func createTopologyIdAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.topology.topologyid"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// NOTE: Topology ID not available via SNMP - generated by management system
	return attr
}

func createTopologyNameAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.topology.name"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// NOTE: Topology name not available via SNMP - configured by management system
	return attr
}

func createTopologyTypeAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.topology.topologytype"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// NOTE: Topology type not available via SNMP - configured by management system
	return attr
}

func createTopologyLastUpdatedAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.topology.lastupdated"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// NOTE: Last updated timestamp managed by topology discovery system
	return attr
}

// Network Node Attributes
func createNetworkNodeIdAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.topology.nodes.nodeid"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// NOTE: Node ID typically derived from device identification, not directly from SNMP
	return attr
}

func createNetworkNodeNameAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.topology.nodes.name"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.1.5.0")) // sysName
	return attr
}

func createNetworkNodeTypeAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.topology.nodes.nodetype"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// NOTE: Node type derived from device classification, not directly from SNMP
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.1.2.0")) // Parse from sysObjectID
	return attr
}

func createNetworkNodeStatusAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.topology.nodes.status"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// NOTE: Node status derived from polling success/failure and device operational state
	return attr
}

func createNetworkNodeLocationAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.topology.nodes.location"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.1.6.0")) // sysLocation
	return attr
}

func createNetworkNodeRegionAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.topology.nodes.region"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// NOTE: Region typically configured or derived from location parsing
	return attr
}

func createNetworkNodeTierAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.topology.nodes.tier"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// NOTE: Tier classification not available via SNMP - network design concept
	return attr
}

// System MIB attribute functions for EquipmentInfo

func createSystemDescription() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.hardware"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.1.1.0")) // sysDescr
	return attr
}

func createSystemUptime() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.uptime", "gpudevice": "gpudevice.deviceinfo.uptime"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.1.3.0")) // sysUpTime
	return attr
}

func createSystemLocation() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.location", "gpudevice": "gpudevice.deviceinfo.location"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.1.6.0")) // sysLocation
	return attr
}

func createSystemDeviceType() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.device_type"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	// Use InferDeviceType rule that analyzes sysObjectID to determine device type
	rule := &l8tpollaris.L8PRule{}
	rule.Name = "InferDeviceType"
	rule.Params = make(map[string]*l8tpollaris.L8PParameter)
	// Pass the sysObjectID OID for analysis
	rule.Params["from"] = &l8tpollaris.L8PParameter{Value: ".1.3.6.1.2.1.1.2.0"} // sysObjectID
	attr.Rules = append(attr.Rules, rule)
	return attr
}

func createSystemSoftware() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.software", "gpudevice": "gpudevice.deviceinfo.osversion"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.1.1.0")) // sysDescr (extract software info)
	return attr
}

func createSystemVersion() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.version", "gpudevice": "gpudevice.deviceinfo.driverversion"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.1.1.0")) // sysDescr (extract version info)
	return attr
}

func createSystemModel() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.model", "gpudevice": "gpudevice.deviceinfo.model"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.1.1.0")) // sysDescr (extract model info)
	return attr
}

func createIpAddress() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.ipaddress", "gpudevice": "gpudevice.deviceinfo.ipaddress"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule("ipaddress"))
	return attr
}

func createDeviceStatus() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.devicestatus", "gpudevice": "gpudevice.deviceinfo.devicestatus"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createDeviceStatusRule())
	return attr
}
