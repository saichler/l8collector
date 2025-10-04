package client

import (
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8types/go/ifs"
)

type RestCollector struct {
	client *RestClient
}

func (this *RestCollector) Init(hostConn *l8tpollaris.L8PHostProtocol, r ifs.IResources) error {
	/*
		clientConfig := &RestClientConfig{
			Host:          hostConn.Addr,
			Port:          hostConn.Port,
			Https:         true,
			TokenRequired: true,
			CertFileName:  hostConn.Cert,
			Prefix:        hostConn.HttpPrefix,
			AuthPaths:     hostConn.AuthPaths,
		}*/
	return nil
}

func (this *RestCollector) Protocol() l8tpollaris.L8PProtocol {
	return l8tpollaris.L8PProtocol_L8PNETCONF
}

func (this *RestCollector) Exec(job *l8tpollaris.CJob) {
}
func (this *RestCollector) Connect() error {
	return nil
}
func (this *RestCollector) Disconnect() error {
	return nil
}
func (this *RestCollector) Online() bool {
	return true
}
