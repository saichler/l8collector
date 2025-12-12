package rest

import (
	"fmt"
	"strings"

	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8web/go/web/client"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type RestCollector struct {
	client       *client.RestClient
	hostProtocol *l8tpollaris.L8PHostProtocol
	resources    ifs.IResources
	connected    bool
}

func (this *RestCollector) Init(hostConn *l8tpollaris.L8PHostProtocol, r ifs.IResources) error {
	clientConfig := &client.RestClientConfig{
		Host:          hostConn.Addr,
		Port:          int(hostConn.Port),
		Https:         true,
		TokenRequired: true,
		CertFileName:  hostConn.Cert,
		Prefix:        hostConn.HttpPrefix,
		AuthInfo: &client.RestAuthInfo{
			NeedAuth:   hostConn.Ainfo.NeedAuth,
			BodyType:   hostConn.Ainfo.AuthBody,
			UserField:  hostConn.Ainfo.AuthUserField,
			PassField:  hostConn.Ainfo.AuthPassField,
			RespType:   hostConn.Ainfo.AuthResp,
			TokenField: hostConn.Ainfo.AuthToken,
			AuthPath:   hostConn.Ainfo.AuthPath,
		},
	}
	client, err := client.NewRestClient(clientConfig, r)
	if err != nil {
		return err
	}
	this.hostProtocol = hostConn
	this.client = client
	this.resources = r
	return nil
}

func (this *RestCollector) Protocol() l8tpollaris.L8PProtocol {
	return l8tpollaris.L8PProtocol_L8PRESTCONF
}

func (this *RestCollector) parseWhat(poll *l8tpollaris.L8Poll) (string, string, proto.Message, error) {
	tokens := strings.Split(poll.What, "::")
	if len(tokens) != 3 {
		return "", "", nil, fmt.Errorf("invalid What format")
	}

	switch tokens[0] {
	case "GET":
		fallthrough
	case "POST":
		fallthrough
	case "PUT":
		fallthrough
	case "PATCH":
		fallthrough
	case "DELETE":
	default:
		return "", "", nil, fmt.Errorf("invalid What method")
	}

	info, err := this.resources.Registry().Info(poll.BodyName)
	if err != nil {
		return "", "", nil, err
	}
	b, _ := info.NewInstance()
	body := b.(proto.Message)

	err = protojson.Unmarshal([]byte(tokens[2]), body)
	if err != nil {
		return "", "", nil, err
	}
	return tokens[0], tokens[1], body, nil
}

func (this *RestCollector) Exec(job *l8tpollaris.CJob) {
	if !this.connected {
		err := this.Connect()
		if err != nil {
			job.ErrorCount++
			job.Error = err.Error()
			return
		}
	}
	poll, err := pollaris.Poll(job.PollarisName, job.JobName, this.resources)
	method, endpoint, body, err := this.parseWhat(poll)
	if err != nil {
		job.ErrorCount++
		job.Error = err.Error()
		return
	}

	resp, err := this.client.Do(method, endpoint, poll.RespName, "", "", body, 1)
	if err != nil {
		job.ErrorCount++
		job.Error = err.Error()
		return
	}

	job.ErrorCount = 0
	job.Result, _ = proto.Marshal(resp)
}

func (this *RestCollector) Connect() error {
	_, user, password, _, err := this.resources.Security().Credential(this.hostProtocol.CredId, "rest", this.resources)
	if err != nil {
		panic(err)
	}
	return this.client.Auth(user, password)
}

func (this *RestCollector) Disconnect() error {
	return nil
}

func (this *RestCollector) Online() bool {
	return true
}
