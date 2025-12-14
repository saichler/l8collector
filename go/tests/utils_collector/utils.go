package utils_collector

import (
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	common2 "github.com/saichler/probler/go/prob/common"
)

const (
	InvServiceName = "NetBox"
	K8sServiceName = "Cluster"
)

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
