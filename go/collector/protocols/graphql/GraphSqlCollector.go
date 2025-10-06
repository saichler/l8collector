package graphql

import (
	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8web/go/web/gclient"
	"google.golang.org/protobuf/proto"
)

type GraphQlCollector struct {
	client       *gclient.GraphQLClient
	hostProtocol *l8tpollaris.L8PHostProtocol
	resources    ifs.IResources
	connected    bool
}

func (this *GraphQlCollector) Init(hostConn *l8tpollaris.L8PHostProtocol, r ifs.IResources) error {
	clientConfig := &gclient.GraphQLClientConfig{
		Host:          hostConn.Addr,
		Port:          int(hostConn.Port),
		Https:         true,
		TokenRequired: false,
		CertFileName:  hostConn.Cert,
		Prefix:        hostConn.HttpPrefix,
		AuthInfo: &gclient.GraphQLAuthInfo{
			NeedAuth:   hostConn.Ainfo.NeedAuth,
			BodyType:   hostConn.Ainfo.AuthBody,
			UserField:  hostConn.Ainfo.AuthUserField,
			PassField:  hostConn.Ainfo.AuthPassField,
			RespType:   hostConn.Ainfo.AuthResp,
			TokenField: hostConn.Ainfo.AuthToken,
			AuthPath:   hostConn.Ainfo.AuthPath,
			ApiKey:     hostConn.Ainfo.ApiKey,
			ApiUser:    hostConn.Ainfo.ApiUser,
			IsAPIKey:   hostConn.Ainfo.IsApiKey,
		},
	}
	client, err := gclient.NewGraphQLClient(clientConfig, r)
	if err != nil {
		return err
	}
	this.hostProtocol = hostConn
	this.client = client
	this.resources = r
	return nil
}

func (this *GraphQlCollector) Protocol() l8tpollaris.L8PProtocol {
	return l8tpollaris.L8PProtocol_L8PGraphQL
}

func (this *GraphQlCollector) Exec(job *l8tpollaris.CJob) {
	if !this.connected {
		err := this.Connect()
		if err != nil {
			job.ErrorCount++
			job.Error = err.Error()
			return
		}
	}
	poll, err := pollaris.Poll(job.PollarisName, job.JobName, this.resources)
	if err != nil {
		job.ErrorCount++
		job.Error = err.Error()
		return
	}

	resp, err := this.client.Query(poll.What, nil, poll.RespName, "")
	if err != nil {
		job.ErrorCount++
		job.Error = err.Error()
		return
	}

	job.ErrorCount = 0
	job.Result, _ = proto.Marshal(resp)
}

func (this *GraphQlCollector) Connect() error {
	return this.client.Auth(this.hostProtocol.Username, this.hostProtocol.Password)
}

func (this *GraphQlCollector) Disconnect() error {
	return nil
}

func (this *GraphQlCollector) Online() bool {
	return true
}
