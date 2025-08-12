package utils_collector

import (
	"github.com/saichler/l8collector/go/collector/common"
	"github.com/saichler/l8pollaris/go/types"
)

const (
	InvServiceName = "NetBox"
	K8sServiceName = "Cluster"
)

func CreateDevice(ip string, serviceArea byte) *types.Device {
	device := &types.Device{}
	device.DeviceId = ip
	device.InventoryService = &types.DeviceServiceInfo{ServiceName: InvServiceName, ServiceArea: int32(serviceArea)}
	device.ParsingService = &types.DeviceServiceInfo{ServiceName: common.ParserServicePrefix + InvServiceName, ServiceArea: int32(serviceArea)}
	device.Hosts = make(map[string]*types.Host)
	host := &types.Host{}
	host.DeviceId = device.DeviceId

	host.Configs = make(map[int32]*types.Connection)
	device.Hosts[device.DeviceId] = host

	sshConfig := &types.Connection{}
	sshConfig.Protocol = types.Protocol_PSSH
	sshConfig.Port = 22
	sshConfig.Addr = ip
	sshConfig.Username = "simadmin"
	sshConfig.Password = "simadmin"
	sshConfig.Terminal = "vt100"
	sshConfig.Timeout = 15

	host.Configs[int32(sshConfig.Protocol)] = sshConfig

	snmpConfig := &types.Connection{}
	snmpConfig.Protocol = types.Protocol_PSNMPV2
	snmpConfig.Addr = ip
	snmpConfig.Port = 161
	snmpConfig.Timeout = 15
	snmpConfig.ReadCommunity = "public"

	host.Configs[int32(snmpConfig.Protocol)] = snmpConfig

	return device
}
