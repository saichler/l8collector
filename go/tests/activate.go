package tests

import (
	"github.com/saichler/l8pollaris/go/pollaris/targets"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/probler/go/prob/common"
)

func ActivateTargets(vnic ifs.IVNic) {
	targets.Activate(common.DB_CREDS, common.DB_NAME, vnic)
}
