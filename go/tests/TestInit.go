package tests

import (
	"github.com/saichler/l8bus/go/overlay/protocol"
	"github.com/saichler/l8pollaris/go/pollaris/targets"
	. "github.com/saichler/l8test/go/infra/t_resources"
	. "github.com/saichler/l8test/go/infra/t_topology"
	. "github.com/saichler/l8types/go/ifs"
	"github.com/saichler/probler/go/prob/common"
)

var topo *TestTopology

func init() {
	Log.SetLogLevel(Trace_Level)
	targets.Links = &common.Links{}
}

func setup() {
	setupTopology()
}

func tear() {
	shutdownTopology()
}

func reset(name string) {
	Log.Info("*** ", name, " end ***")
	topo.ResetHandlers()
}

func setupTopology() {
	protocol.MessageLog = true
	topo = NewTestTopology(4, []int{20000, 30000, 40000}, Info_Level)
}

func shutdownTopology() {
	topo.Shutdown()
}
