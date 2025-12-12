package common

import (
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8types/go/ifs"
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
	Init(*l8tpollaris.L8PHostProtocol, ifs.IResources) error
	Protocol() l8tpollaris.L8PProtocol
	Exec(job *l8tpollaris.CJob)
	Connect() error
	Disconnect() error
	Online() bool
}

var SmoothFirstCollection = false
