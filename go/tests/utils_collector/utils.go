package utils_collector

import (
	"fmt"
	"github.com/saichler/l8parser/go/parser/boot"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8types/go/ifs"
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
	host.HostId = device.TargetId

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
	host.HostId = device.TargetId

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

func SetPolls(sla *ifs.ServiceLevelAgreement) {
	initData := []interface{}{}
	for _, p := range boot.GetAllPolarisModels() {
		for _, poll := range p.Polling {
			fmt.Println(p.Name, "/", poll.Name)
			if poll.Cadence.Enabled {
				poll.Cadence.Cadences[0] = 3
			}
			initData = append(initData, p)
		}
	}
	initData = append(initData, boot.CreateK8sBootPolls())
	sla.SetInitItems(initData)
}
