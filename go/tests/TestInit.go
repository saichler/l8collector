/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
You may obtain a copy of the License at:

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package tests provides integration tests for the L8Collector service.
// Tests verify the collection functionality using a multi-node test topology
// with simulated devices and protocol collectors.
//
// The test framework uses:
//   - TestTopology: Multi-node virtual network for service testing
//   - MockParserService: Receives and validates collected data
//   - Protocol simulators: Mock SNMP, REST, GraphQL endpoints
package tests

import (
	"github.com/saichler/l8bus/go/overlay/protocol"
	"github.com/saichler/l8pollaris/go/pollaris/targets"
	. "github.com/saichler/l8test/go/infra/t_resources"
	. "github.com/saichler/l8test/go/infra/t_topology"
	. "github.com/saichler/l8types/go/ifs"
	"github.com/saichler/probler/go/prob/common"
)

// topo is the shared test topology instance used across all tests.
var topo *TestTopology

// init sets up the test environment with trace-level logging
// and initializes the targets.Links registry.
func init() {
	Log.SetLogLevel(Trace_Level)
	targets.Links = &common.Links{}
}

// setup initializes the test topology before tests run.
func setup() {
	setupTopology()
}

// tear shuts down the test topology after tests complete.
func tear() {
	shutdownTopology()
}

// reset cleans up after a test by logging completion and resetting handlers.
func reset(name string) {
	Log.Info("*** ", name, " end ***")
	topo.ResetHandlers()
}

// setupTopology creates a 4-node test topology with message logging enabled.
func setupTopology() {
	protocol.MessageLog = true
	topo = NewTestTopology(4, []int{20000, 30000, 40000}, Info_Level)
}

// shutdownTopology gracefully shuts down the test topology.
func shutdownTopology() {
	topo.Shutdown()
}
