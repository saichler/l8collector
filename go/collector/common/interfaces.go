package common

import (
	"github.com/saichler/collect/go/types"
	"github.com/saichler/l8types/go/ifs"
)

const (
	CollectorService    = "Collector"
	ParserServicePrefix = "P-"
)

const (
	BOOT_STAGE_00 = "Boot_Stage_00"
	BOOT_STAGE_01 = "Boot_Stage_01"
	BOOT_STAGE_02 = "Boot_Stage_02"
	BOOT_STAGE_03 = "Boot_Stage_03"
	BOOT_STAGE_04 = "Boot_Stage_04"
)

var BootStages = []string{BOOT_STAGE_00, BOOT_STAGE_01, BOOT_STAGE_02, BOOT_STAGE_03, BOOT_STAGE_04}

type ProtocolCollector interface {
	Init(*l8poll.L8T_Connection, ifs.IResources) error
	Protocol() types.Protocol
	Exec(job *l8poll.CJob)
	Connect() error
	Disconnect() error
	Online() bool
}

var SmoothFirstCollection = false
