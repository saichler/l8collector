package utils_collector

import (
	"github.com/saichler/l8collector/go/collector/common"
	"github.com/saichler/l8pollaris/go/types/l8poll"
)

const (
	InvServiceName = "NetBox"
	K8sServiceName = "Cluster"
)

func CreateDevice(ip string, serviceArea byte) *l8poll.L8C_Target {
	device := &l8poll.L8C_Target{}
	device.TargetId = ip
	device.InventoryService = &l8poll.L8ServiceInfo{ServiceName: InvServiceName, ServiceArea: int32(serviceArea)}
	device.ParsingService = &l8poll.L8ServiceInfo{ServiceName: common.ParserServicePrefix + InvServiceName, ServiceArea: int32(serviceArea)}
	device.Hosts = make(map[string]*l8poll.L8C_Host)
	host := &l8poll.L8C_Host{}
	host.TargetId = device.TargetId

	host.Configs = make(map[int32]*l8poll.L8T_Connection)
	device.Hosts[device.TargetId] = host

	sshConfig := &l8poll.L8T_Connection{}
	sshConfig.Protocol = l8poll.L8C_Protocol_L8P_SSH
	sshConfig.Port = 22
	sshConfig.Addr = ip
	sshConfig.Username = "simadmin"
	sshConfig.Password = "simadmin"
	sshConfig.Terminal = "vt100"
	sshConfig.Timeout = 15

	host.Configs[int32(sshConfig.Protocol)] = sshConfig

	snmpConfig := &l8poll.L8T_Connection{}
	snmpConfig.Protocol = l8poll.L8C_Protocol_L8P_PSNMPV2
	snmpConfig.Addr = ip
	snmpConfig.Port = 161
	snmpConfig.Timeout = 15
	snmpConfig.ReadCommunity = "public"

	host.Configs[int32(snmpConfig.Protocol)] = snmpConfig

	return device
}
