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

package tests

import (
	"fmt"
	targets2 "github.com/saichler/l8pollaris/go/pollaris/targets"
	common2 "github.com/saichler/probler/go/prob/common"
	"github.com/saichler/probler/go/prob/common/creates"
	"testing"
	"time"

	"github.com/saichler/l8collector/go/collector/service"
	"github.com/saichler/l8collector/go/tests/utils_collector"
	"github.com/saichler/l8parser/go/parser/boot"
	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
)

// TestMain is the test suite entry point that manages the test topology lifecycle.
// It calls setup() before any tests run and tear() after all tests complete.
func TestMain(m *testing.M) {
	setup()
	m.Run()
	tear()
}

// TestCollector is the main integration test for the collector service.
// It sets up a complete test environment with:
//   - Pollaris service for poll configuration
//   - CollectorService for data collection
//   - MockParsingService to receive and validate collected data
//
// The test creates a simulated device (using opensim) and verifies that
// all configured jobs execute exactly once during the boot sequence.
// It also tests the on-demand job execution via the ExecuteService.
//
// Prerequisites:
//   - opensim running with simulated devices
//   - Example: curl -X POST http://localhost:8080/api/v1/devices -H "Content-Type: application/json" -d '{"start_ip":"10.10.10.1","device_count":3,"netmask":"24"}'
func TestCollector(t *testing.T) {

	cServiceName, cServiceArea := targets2.Links.Collector(common2.NetworkDevice_Links_ID)
	pServiceName, pServiceArea := targets2.Links.Parser(common2.NetworkDevice_Links_ID)

	//use opensim to simulate this device with this ip
	//https://github.com/saichler/opensim
	//curl -X POST http://localhost:8080/api/v1/devices -H "Content-Type: application/json" -d '{"start_ip":"10.10.10.1","device_count":3,"netmask":"24"}'
	device := creates.CreateDevice("10.20.30.1", common2.NetworkDevice_Links_ID, "sim")

	vnic := topo.VnicByVnetNum(2, 2)
	sla := ifs.NewServiceLevelAgreement(&pollaris.PollarisService{}, pollaris.ServiceName, pollaris.ServiceArea, true, nil)
	utils_collector.SetPolls(sla)
	vnic.Resources().Services().Activate(sla, vnic)

	ActivateTargets(vnic)

	sla = ifs.NewServiceLevelAgreement(&service.CollectorService{}, cServiceName, cServiceArea, true, nil)
	vnic.Resources().Services().Activate(sla, vnic)

	sla = ifs.NewServiceLevelAgreement(&utils_collector.MockParsingService{}, pServiceName, pServiceArea, false, nil)
	vnic.Resources().Services().Activate(sla, vnic)

	time.Sleep(time.Second)

	/*
		defer func() {
			deActivateDeviceAndPollConfigServices(cfg, 0)
		}()
	*/

	cl := topo.VnicByVnetNum(1, 1)
	err := cl.Multicast(cServiceName, cServiceArea, ifs.POST, device)
	if err != nil {
		panic(err)
	}

	time.Sleep(time.Second * 3)

	mp, ok := vnic.Resources().Services().ServiceHandler(pServiceName, pServiceArea)
	if !ok {
		panic("No mock service found")
	}
	mock := mp.(*utils_collector.MockParsingService)
	for k, v := range mock.JobsCounts() {
		for k1, v1 := range v {
			if v1 != 1 {
				vnic.Resources().Logger().Fail(t, "Expected 1 but got ", v1, " job ", k, ":", k1)
			}
		}
	}

	job := &l8tpollaris.CJob{}
	job.TargetId = device.TargetId
	job.HostId = device.TargetId
	job.PollarisName = "boot03"
	job.JobName = "entityMib"

	exec := service.Exec(cServiceArea, vnic.Resources())
	ob := object.New(nil, job)
	exec.Post(ob, vnic)
	fmt.Println(job.Result)
}

// testJobDisable tests the job disabling functionality.
// It modifies poll cadences to run more frequently and changes
// the entityMib OID to test error handling when a job fails.
// This test is not exported (lowercase) and may be used for manual testing.
func testJobDisable(t *testing.T) {
	cServiceName, cServiceArea := targets2.Links.Collector(common2.NetworkDevice_Links_ID)
	pServiceName, pServiceArea := targets2.Links.Parser(common2.NetworkDevice_Links_ID)

	snmpPolls := boot.GetAllPolarisModels()
	for _, snmpPoll := range snmpPolls {
		for _, poll := range snmpPoll.Polling {
			if poll.Cadence.Enabled {
				poll.Cadence.Cadences[0] = 3
			}
			if poll.Name == "entityMib" {
				poll.What = ".1.3.6.6.6"
			}
		}
	}

	//use opensim to simulate this device with this ip
	//https://github.com/saichler/opensim
	//curl -X POST http://localhost:8080/api/v1/devices -H "Content-Type: application/json" -d '{"start_ip":"10.10.10.1","device_count":3,"netmask":"24"}'
	device := creates.CreateDevice("10.20.30.1", common2.NetworkDevice_Links_ID, "sim")

	vnic := topo.VnicByVnetNum(2, 2)
	sla := ifs.NewServiceLevelAgreement(&pollaris.PollarisService{}, pollaris.ServiceName, pollaris.ServiceArea, true, nil)
	vnic.Resources().Services().Activate(sla, vnic)

	ActivateTargets(vnic)

	sla = ifs.NewServiceLevelAgreement(&service.CollectorService{}, cServiceName, cServiceArea, true, nil)
	vnic.Resources().Services().Activate(sla, vnic)

	sla = ifs.NewServiceLevelAgreement(&utils_collector.MockParsingService{}, pServiceName, pServiceArea, false, nil)
	vnic.Resources().Services().Activate(sla, vnic)

	time.Sleep(time.Second)

	p := pollaris.Pollaris(vnic.Resources())
	for _, poll := range snmpPolls {
		err := p.Post(poll, false)
		if err != nil {
			vnic.Resources().Logger().Fail(t, err.Error())
			return
		}
	}

	cl := topo.VnicByVnetNum(1, 1)
	err := cl.Multicast(targets2.ServiceName, targets2.ServiceArea, ifs.POST, device)
	if err != nil {
		panic(err)
	}

	time.Sleep(time.Second * 300)

}
