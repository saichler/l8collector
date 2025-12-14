package tests

import (
	"fmt"
	"github.com/saichler/l8parser/go/parser/boot"
	"github.com/saichler/l8pollaris/go/pollaris/targets"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/probler/go/prob/common"
)

func ActivateTargets(vnic ifs.IVNic) {
	targets.Activate(common.DB_CREDS, common.DB_NAME, vnic)
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
