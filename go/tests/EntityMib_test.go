package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/saichler/l8collector/go/collector/common"
	"github.com/saichler/l8collector/go/collector/devices"
	"github.com/saichler/l8collector/go/collector/service"
	"github.com/saichler/l8collector/go/tests/utils_collector"
	"github.com/saichler/l8parser/go/parser/boot"
	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
)

func TestEntityMib(t *testing.T) {
	serviceArea := byte(0)
	snmpPolls := boot.GetAllPolarisModels()
	for _, snmpPoll := range snmpPolls {
		for _, poll := range snmpPoll.Polling {
			if poll.Cadence.Enabled {
				poll.Cadence.Cadences[0] = 3
			}
		}
	}

	//use opensim to simulate this device with this ip
	//https://github.com/saichler/opensim
	//curl -X POST http://localhost:8080/api/v1/devices -H "Content-Type: application/json" -d '{"start_ip":"10.10.10.1","device_count":3,"netmask":"24"}'
	device := utils_collector.CreateDevice("10.20.30.3", serviceArea)

	vnic := topo.VnicByVnetNum(2, 2)
	vnic.Resources().Registry().Register(pollaris.PollarisService{})
	vnic.Resources().Services().Activate(pollaris.ServiceType, pollaris.ServiceName, serviceArea, vnic.Resources(), vnic)
	vnic.Resources().Registry().Register(targets.DeviceService{})
	vnic.Resources().Services().Activate(targets.ServiceType, targets.ServiceName, serviceArea, vnic.Resources(), vnic)
	vnic.Resources().Registry().Register(service.CollectorService{})
	vnic.Resources().Services().Activate(service.ServiceType, common.CollectorService, serviceArea, vnic.Resources(), vnic)

	vnic.Resources().Registry().Register(utils_collector.MockParsingService{})
	vnic.Resources().Services().Activate(utils_collector.ServiceType, device.ParsingService.ServiceName, byte(device.ParsingService.ServiceArea),
		vnic.Resources(), vnic)

	time.Sleep(time.Second)

	p := pollaris.Pollaris(vnic.Resources())
	for _, poll := range snmpPolls {
		err := p.Add(poll, false)
		if err != nil {
			vnic.Resources().Logger().Fail(t, err.Error())
			return
		}
	}

	cl := topo.VnicByVnetNum(1, 1)
	cl.Multicast(targets.ServiceName, serviceArea, ifs.POST, device)

	time.Sleep(time.Second)

	job := &types.CJob{}
	job.DeviceId = device.DeviceId
	job.HostId = device.DeviceId
	job.PollarisName = "mib2"
	job.JobName = "entityMib"

	exec := service.Exec(serviceArea, vnic.Resources())
	ob := object.New(nil, job)
	exec.Post(ob, vnic)
	fmt.Println(job.Result)
}
