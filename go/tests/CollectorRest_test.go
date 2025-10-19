package tests

import (
	"testing"
	"time"

	"github.com/saichler/l8collector/go/collector/common"
	"github.com/saichler/l8collector/go/collector/service"
	"github.com/saichler/l8collector/go/collector/targets"
	"github.com/saichler/l8collector/go/tests/utils_collector"
	"github.com/saichler/l8parser/go/parser/boot"
	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
)

func TestRestCollector(t *testing.T) {

	p := &l8tpollaris.L8Pollaris{}
	p.Groups = []string{common.BOOT_STAGE_00}
	p.Name = "devices"

	poll := &l8tpollaris.L8Poll{}
	poll.What = "GET::/probler/0/NetDev::{\"text\":\"select * from networkdevice where networkdevice.id=10.20.30.1\"}"
	poll.BodyName = "L8Query"
	poll.Name = "devices"
	poll.Cadence = boot.EVERY_5_MINUTES
	poll.Protocol = l8tpollaris.L8PProtocol_L8PRESTCONF
	p.Polling = map[string]*l8tpollaris.L8Poll{poll.Name: poll}

	host := utils_collector.CreateRestHost("192.168.86.226", 2443, "admin", "Admin123!")

	serviceArea := byte(0)

	vnic := topo.VnicByVnetNum(2, 2)
	sla := ifs.NewServiceLevelAgreement(&pollaris.PollarisService{}, pollaris.ServiceName, serviceArea, true, nil)
	vnic.Resources().Services().Activate(sla, vnic)

	sla = ifs.NewServiceLevelAgreement(&targets.TargetService{}, targets.ServiceName, serviceArea, true, nil)
	vnic.Resources().Services().Activate(sla, vnic)

	sla = ifs.NewServiceLevelAgreement(&service.CollectorService{}, common.CollectorService, serviceArea, true, nil)
	vnic.Resources().Services().Activate(sla, vnic)

	sla = ifs.NewServiceLevelAgreement(&utils_collector.MockParsingService{}, host.LinkParser.ZsideServiceName,
		byte(host.LinkParser.ZsideServiceArea), false, nil)
	vnic.Resources().Services().Activate(sla, vnic)

	pollaris.Pollaris(vnic.Resources()).Post(p, true)
	vnic.Resources().Registry().Register(&l8api.AuthUser{})
	vnic.Resources().Registry().Register(&l8api.AuthToken{})
	vnic.Resources().Registry().Register(l8api.L8Query{})

	time.Sleep(time.Second)

	cl := topo.VnicByVnetNum(1, 1)
	err := cl.Multicast(targets.ServiceName, serviceArea, ifs.POST, host)
	if err != nil {
		panic(err)
	}

	time.Sleep(time.Second * 10)
}
