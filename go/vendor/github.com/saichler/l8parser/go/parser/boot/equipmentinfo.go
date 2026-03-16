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

// equipmentinfo.go provides attribute functions for previously unpopulated
// NetworkDevice.EquipmentInfo fields. These are wired into Boot01/Boot03
// generic polls and vendor-specific polls.
package boot

import (
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
)

// createFirmwareVersionAttribute polls entPhysicalSoftwareRev (chassis entry)
// from the standard ENTITY-MIB (RFC 4133).
func createFirmwareVersionAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.firmwareversion"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.10.1")) // entPhysicalSoftwareRev
	return attr
}

// createGenericSerialAttribute polls entPhysicalSerialNum (chassis entry)
// from the standard ENTITY-MIB. Used as a generic fallback for vendors
// without a vendor-specific serial number OID.
func createGenericSerialAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.serialnumber"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.11.1")) // entPhysicalSerialNum
	return attr
}

// createInterfaceCountAttribute polls ifNumber from IF-MIB,
// which returns the total number of network interfaces on the device.
func createInterfaceCountAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.interfacecount"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.2.1.0")) // ifNumber
	return attr
}

// createVendorTypeOidAttribute polls entPhysicalVendorType (chassis entry)
// from the standard ENTITY-MIB (RFC 2737).
func createVendorTypeOidAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.vendortypeoid"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.3.1")) // entPhysicalVendorType
	return attr
}

// createPhysicalAliasAttribute polls entPhysicalAlias (chassis entry)
// from the standard ENTITY-MIB (RFC 2737).
func createPhysicalAliasAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.physicalalias"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.14.1")) // entPhysicalAlias
	return attr
}

// createAssetIdAttribute polls entPhysicalAssetID (chassis entry)
// from the standard ENTITY-MIB (RFC 2737).
func createAssetIdAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.assetid"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.15.1")) // entPhysicalAssetID
	return attr
}

// createIsFruAttribute polls entPhysicalIsFRU (chassis entry)
// from the standard ENTITY-MIB (RFC 2737).
// Note: SNMP TruthValue 1=true, 2=false.
func createIsFruAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.isfru"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.16.1")) // entPhysicalIsFRU
	return attr
}

// createManufacturingDateAttribute polls entPhysicalMfgDate (chassis entry)
// from the standard ENTITY-MIB (RFC 4133).
// Note: DateAndTime SNMP type (OCTET STRING, 8 or 11 bytes).
func createManufacturingDateAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.manufacturingdate"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.17.1")) // entPhysicalMfgDate
	return attr
}

// createManufacturerNameAttribute polls entPhysicalMfgName (chassis entry)
// from the standard ENTITY-MIB (RFC 4133).
func createManufacturerNameAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.manufacturername"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.18.1")) // entPhysicalMfgName
	return attr
}

// createIdentificationUrisAttribute polls entPhysicalUris (chassis entry)
// from the standard ENTITY-MIB (RFC 4133).
// Contains newline-separated URIs (e.g., CLEI codes).
func createIdentificationUrisAttribute() *l8tpollaris.L8PAttribute {
	attr := &l8tpollaris.L8PAttribute{}
	attr.PropertyId = map[string]string{"networkdevice": "networkdevice.equipmentinfo.identificationuris"}
	attr.Rules = make([]*l8tpollaris.L8PRule, 0)
	attr.Rules = append(attr.Rules, createSetRule(".1.3.6.1.2.1.47.1.1.1.1.19.1")) // entPhysicalUris
	return attr
}
