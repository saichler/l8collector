package common

import (
	"github.com/saichler/l8pollaris/go/types"
	"github.com/saichler/l8types/go/ifs"
)

const (
	CollectorService    = "Collector"
	ParserServicePrefix = "P-"
	BOOT_GROUP          = "BOOT"
)

type ProtocolCollector interface {
	Init(*types.Connection, ifs.IResources) error
	Protocol() types.Protocol
	Exec(*types.CJob)
	Connect() error
	Disconnect() error
	Online() bool
}
