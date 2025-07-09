package tests

import (
	"github.com/saichler/l8collector/go/collector/devices"
	"github.com/saichler/l8pollaris/go/pollaris"
	"testing"
)

func TestMain(m *testing.M) {
	setup()
	m.Run()
	tear()
}

func TestPollaris(t *testing.T) {
	vnic := topo.VnicByVnetNum(2, 2)
	vnic.Resources().Registry().Register(pollaris.PollarisService{})
	vnic.Resources().Services().Activate(pollaris.ServiceType, pollaris.ServiceName, 0, vnic.Resources(), vnic)
	vnic.Resources().Registry().Register(devices.DeviceService{})
	vnic.Resources().Services().Activate(devices.ServiceType, pollaris.ServiceName, 0, vnic.Resources(), vnic)
}
