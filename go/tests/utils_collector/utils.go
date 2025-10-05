package utils_collector

import (
	"encoding/base64"
	"os"

	"github.com/saichler/l8collector/go/collector/common"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8types/go/types/l8services"
)

const (
	InvServiceName = "NetBox"
	K8sServiceName = "Cluster"
)

func CreateDevice(ip string, serviceArea byte) *l8tpollaris.L8PTarget {
	device := &l8tpollaris.L8PTarget{}
	device.TargetId = ip
	device.LinkData = &l8services.L8ServiceLink{ZsideServiceName: InvServiceName, ZsideServiceArea: int32(serviceArea)}
	device.LinkParser = &l8services.L8ServiceLink{ZsideServiceName: common.ParserServicePrefix + InvServiceName, ZsideServiceArea: int32(serviceArea)}
	device.Hosts = make(map[string]*l8tpollaris.L8PHost)
	host := &l8tpollaris.L8PHost{}
	host.TargetId = device.TargetId

	host.Configs = make(map[int32]*l8tpollaris.L8PHostProtocol)
	device.Hosts[device.TargetId] = host

	sshConfig := &l8tpollaris.L8PHostProtocol{}
	sshConfig.Protocol = l8tpollaris.L8PProtocol_L8PSSH
	sshConfig.Port = 22
	sshConfig.Addr = ip
	sshConfig.Username = "simadmin"
	sshConfig.Password = "simadmin"
	sshConfig.Terminal = "vt100"
	sshConfig.Timeout = 15

	host.Configs[int32(sshConfig.Protocol)] = sshConfig

	snmpConfig := &l8tpollaris.L8PHostProtocol{}
	snmpConfig.Protocol = l8tpollaris.L8PProtocol_L8PPSNMPV2
	snmpConfig.Addr = ip
	snmpConfig.Port = 161
	snmpConfig.Timeout = 15
	snmpConfig.ReadCommunity = "public"

	host.Configs[int32(snmpConfig.Protocol)] = snmpConfig

	return device
}

func CreateCluster(kubeconfig, context string, serviceArea int32) *l8tpollaris.L8PTarget {
	device := &l8tpollaris.L8PTarget{}
	device.TargetId = context
	device.LinkData = &l8services.L8ServiceLink{ZsideServiceName: K8sServiceName, ZsideServiceArea: serviceArea}
	device.LinkParser = &l8services.L8ServiceLink{ZsideServiceName: common.ParserServicePrefix + K8sServiceName, ZsideServiceArea: int32(serviceArea)}
	device.Hosts = make(map[string]*l8tpollaris.L8PHost)
	host := &l8tpollaris.L8PHost{}
	host.TargetId = device.TargetId

	host.Configs = make(map[int32]*l8tpollaris.L8PHostProtocol)
	device.Hosts[device.TargetId] = host

	k8sConfig := &l8tpollaris.L8PHostProtocol{}

	data, err := os.ReadFile(kubeconfig)
	if err != nil {
		panic(err)
	}
	k8sConfig.KubeConfig = base64.StdEncoding.EncodeToString(data)

	k8sConfig.KukeContext = context
	k8sConfig.Protocol = l8tpollaris.L8PProtocol_L8PKubectl

	host.Configs[int32(k8sConfig.Protocol)] = k8sConfig

	return device

	return nil
}

func CreateHost(addr string, port int, user, pass string) *l8tpollaris.L8PTarget {
	device := &l8tpollaris.L8PTarget{}
	device.TargetId = addr
	device.LinkData = &l8services.L8ServiceLink{ZsideServiceName: InvServiceName, ZsideServiceArea: int32(0)}
	device.LinkParser = &l8services.L8ServiceLink{ZsideServiceName: common.ParserServicePrefix + InvServiceName, ZsideServiceArea: int32(0)}
	device.Hosts = make(map[string]*l8tpollaris.L8PHost)
	host := &l8tpollaris.L8PHost{}
	host.TargetId = device.TargetId

	host.Configs = make(map[int32]*l8tpollaris.L8PHostProtocol)
	device.Hosts[device.TargetId] = host

	restConfig := &l8tpollaris.L8PHostProtocol{}
	restConfig.Port = int32(port)
	restConfig.Addr = addr
	restConfig.Username = user
	restConfig.Password = pass
	restConfig.Protocol = l8tpollaris.L8PProtocol_L8PRESTCONF
	restConfig.Timeout = 30

	restConfig.Ainfo = &l8tpollaris.AuthInfo{
		NeedAuth:      true,
		AuthBody:      "AuthUser",
		AuthResp:      "AuthToken",
		AuthUserField: "User",
		AuthPassField: "Pass",
		AuthPath:      "/auth",
		AuthToken:     "Token",
	}

	host.Configs[int32(restConfig.Protocol)] = restConfig

	return device
}
