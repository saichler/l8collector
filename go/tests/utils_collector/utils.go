package utils_collector

import (
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	common2 "github.com/saichler/probler/go/prob/common"
)

const (
	InvServiceName = "NetBox"
	K8sServiceName = "Cluster"
)

func CreateDevice(ip string, serviceArea byte) *l8tpollaris.L8PTarget {
	device := &l8tpollaris.L8PTarget{}
	device.TargetId = ip
	device.LinksId = common2.NetworkDevice_Links_ID
	device.Hosts = make(map[string]*l8tpollaris.L8PHost)
	host := &l8tpollaris.L8PHost{}
	host.TargetId = device.TargetId

	host.Configs = make(map[int32]*l8tpollaris.L8PHostProtocol)
	device.Hosts[device.TargetId] = host

	sshConfig := &l8tpollaris.L8PHostProtocol{}
	sshConfig.Protocol = l8tpollaris.L8PProtocol_L8PSSH
	sshConfig.Port = 22
	sshConfig.Addr = ip
	sshConfig.CredId = "sim"
	sshConfig.Terminal = "vt100"
	sshConfig.Timeout = 15

	host.Configs[int32(sshConfig.Protocol)] = sshConfig

	snmpConfig := &l8tpollaris.L8PHostProtocol{}
	snmpConfig.Protocol = l8tpollaris.L8PProtocol_L8PPSNMPV2
	snmpConfig.Addr = ip
	snmpConfig.Port = 161
	snmpConfig.Timeout = 15
	snmpConfig.CredId = "sim"

	host.Configs[int32(snmpConfig.Protocol)] = snmpConfig

	return device
}

func CreateCluster(kubeconfig, context string, serviceArea int32) *l8tpollaris.L8PTarget {
	device := &l8tpollaris.L8PTarget{}
	device.TargetId = context
	device.LinksId = common2.K8s_Links_ID
	device.Hosts = make(map[string]*l8tpollaris.L8PHost)
	host := &l8tpollaris.L8PHost{}
	host.TargetId = device.TargetId

	host.Configs = make(map[int32]*l8tpollaris.L8PHostProtocol)
	device.Hosts[device.TargetId] = host

	k8sConfig := &l8tpollaris.L8PHostProtocol{}

	k8sConfig.CredId = "lab"
	k8sConfig.Protocol = l8tpollaris.L8PProtocol_L8PKubectl

	host.Configs[int32(k8sConfig.Protocol)] = k8sConfig

	return device
}

func CreateRestHost(addr string, port int, user, pass string) *l8tpollaris.L8PTarget {
	device := &l8tpollaris.L8PTarget{}
	device.TargetId = addr
	device.LinksId = common2.NetworkDevice_Links_ID
	device.Hosts = make(map[string]*l8tpollaris.L8PHost)
	host := &l8tpollaris.L8PHost{}
	host.TargetId = device.TargetId

	host.Configs = make(map[int32]*l8tpollaris.L8PHostProtocol)
	device.Hosts[device.TargetId] = host

	restConfig := &l8tpollaris.L8PHostProtocol{}
	restConfig.Port = int32(port)
	restConfig.Addr = addr
	restConfig.CredId = "sim"
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

func CreateGraphqlHost(addr string, port int, user, pass string) *l8tpollaris.L8PTarget {
	device := &l8tpollaris.L8PTarget{}
	device.TargetId = addr
	device.LinksId = common2.NetworkDevice_Links_ID
	device.Hosts = make(map[string]*l8tpollaris.L8PHost)
	host := &l8tpollaris.L8PHost{}
	host.TargetId = device.TargetId

	host.Configs = make(map[int32]*l8tpollaris.L8PHostProtocol)
	device.Hosts[device.TargetId] = host

	graphQlConfig := &l8tpollaris.L8PHostProtocol{}
	graphQlConfig.Port = int32(port)
	graphQlConfig.Addr = addr
	graphQlConfig.CredId = "sim"
	graphQlConfig.Protocol = l8tpollaris.L8PProtocol_L8PGraphQL
	graphQlConfig.Timeout = 30

	graphQlConfig.Ainfo = &l8tpollaris.AuthInfo{
		NeedAuth: false,
		IsApiKey: true,
		ApiUser:  user,
		ApiKey:   pass,
	}

	host.Configs[int32(graphQlConfig.Protocol)] = graphQlConfig

	return device
}
