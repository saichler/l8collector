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
	targets2 "github.com/saichler/l8pollaris/go/pollaris/targets"
	common2 "github.com/saichler/probler/go/prob/common"
	"os"
	"testing"
	"time"

	"github.com/saichler/l8collector/go/collector/common"
	"github.com/saichler/l8collector/go/collector/service"
	"github.com/saichler/l8collector/go/tests/utils_collector"
	"github.com/saichler/l8parser/go/parser/boot"
	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
	"github.com/saichler/podys/go/types/taddy"
)

// TestGraphqlCollector tests the GraphQL protocol collector functionality.
// It creates a pollaris configuration for querying a GraphQL API (Taddy podcast service)
// and verifies that the collector can successfully execute GraphQL queries.
//
// Environment Variables Required:
//   - X_USER_ID: API user ID for Taddy service
//   - X_API_KEY: API key for Taddy service
//
// The test sets up:
//   - A GraphQL host configuration with API key authentication
//   - Poll configuration for podcast series queries
//   - CollectorService and MockParsingService for validation
func TestGraphqlCollector(t *testing.T) {

	cServiceName, cServiceArea := targets2.Links.Collector(common2.NetworkDevice_Links_ID)
	pServiceName, pServiceArea := targets2.Links.Parser(common2.NetworkDevice_Links_ID)

	p := &l8tpollaris.L8Pollaris{}
	p.Groups = []string{common.BOOT_STAGE_00}
	p.Name = "podcast"

	poll := &l8tpollaris.L8Poll{}
	poll.What = "query { getPodcastSeries(name:\"The Daily\"){ uuid name itunesId description imageUrl totalEpisodesCount itunesInfo{ uuid baseArtworkUrlOf(size:640)}}}"
	poll.Name = "podcast"
	poll.Cadence = boot.EVERY_5_MINUTES
	poll.Protocol = l8tpollaris.L8PProtocol_L8PGraphQL
	poll.RespName = "TaddyResponse"
	p.Polling = map[string]*l8tpollaris.L8Poll{poll.Name: poll}

	host := utils_collector.CreateGraphqlHost("api.taddy.org", 443, os.Getenv("X_USER_ID"), os.Getenv("X_API_KEY"))

	vnic := topo.VnicByVnetNum(2, 2)
	vnic.Resources().Registry().Register(&taddy.TaddyResponse{})

	sla := ifs.NewServiceLevelAgreement(&pollaris.PollarisService{}, pollaris.ServiceName, pollaris.ServiceArea, true, nil)
	vnic.Resources().Services().Activate(sla, vnic)

	ActivateTargets(vnic)

	sla = ifs.NewServiceLevelAgreement(&service.CollectorService{}, cServiceName, cServiceArea, true, nil)
	vnic.Resources().Services().Activate(sla, vnic)

	sla = ifs.NewServiceLevelAgreement(&utils_collector.MockParsingService{}, pServiceName, pServiceArea, false, nil)
	vnic.Resources().Services().Activate(sla, vnic)

	pollaris.Pollaris(vnic.Resources()).Post(p, true)
	vnic.Resources().Registry().Register(&l8api.AuthUser{})
	vnic.Resources().Registry().Register(&l8api.AuthToken{})
	vnic.Resources().Registry().Register(l8api.L8Query{})

	time.Sleep(time.Second)

	cl := topo.VnicByVnetNum(1, 1)
	err := cl.Multicast(targets2.ServiceName, targets2.ServiceArea, ifs.POST, host)
	if err != nil {
		panic(err)
	}

	time.Sleep(time.Second * 10)
}
