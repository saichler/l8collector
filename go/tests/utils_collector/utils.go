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

package utils_collector

import (
	"fmt"
	"github.com/saichler/l8parser/go/parser/boot"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8types/go/ifs"
	common2 "github.com/saichler/probler/go/prob/common"
)

// Service name constants for test configurations.
const (
	// InvServiceName is the service name for inventory/NetBox testing.
	InvServiceName = "NetBox"
	// K8sServiceName is the service name for Kubernetes cluster testing.
	K8sServiceName = "Cluster"
)

// CreateRestHost creates a test target configured for REST/RESTCONF protocol collection.
// It sets up a target with username/password authentication for REST API access.
//
// Parameters:
//   - addr: The IP address or hostname of the REST API endpoint
//   - port: The port number for the REST API (typically 443 for HTTPS)
//   - user: The username for authentication
//   - pass: The password for authentication
//
// Returns:
//   - A configured L8PTarget ready for REST collection testing
func CreateRestHost(addr string, port int, user, pass string) *l8tpollaris.L8PTarget {
	device := &l8tpollaris.L8PTarget{}
	device.TargetId = addr
	device.LinksId = common2.NetworkDevice_Links_ID
	device.Hosts = make(map[string]*l8tpollaris.L8PHost)
	host := &l8tpollaris.L8PHost{}
	host.HostId = device.TargetId

	host.Configs = make(map[int32]*l8tpollaris.L8PHostProtocol)
	device.Hosts[device.TargetId] = host

	restConfig := &l8tpollaris.L8PHostProtocol{}
	restConfig.Port = int32(port)
	restConfig.Addr = addr
	restConfig.CredId = "sim"
	restConfig.Protocol = l8tpollaris.L8PProtocol_L8PRESTCONF
	restConfig.Timeout = 30

	restConfig.Ainfo = &l8tpollaris.AuthInfo{
		NeedAuth:      true,
		AuthBody:      "AuthUser",
		AuthResp:      "AuthToken",
		AuthUserField: "User",
		AuthPassField: "Pass",
		AuthPath:      "/auth",
		AuthToken:     "Token",
	}

	host.Configs[int32(restConfig.Protocol)] = restConfig

	return device
}

// CreateGraphqlHost creates a test target configured for GraphQL protocol collection.
// It sets up a target with API key authentication for GraphQL API access.
//
// Parameters:
//   - addr: The hostname of the GraphQL API endpoint
//   - port: The port number for the GraphQL API (typically 443 for HTTPS)
//   - user: The API user ID (X-USER-ID header)
//   - pass: The API key (X-API-KEY header)
//
// Returns:
//   - A configured L8PTarget ready for GraphQL collection testing
func CreateGraphqlHost(addr string, port int, user, pass string) *l8tpollaris.L8PTarget {
	device := &l8tpollaris.L8PTarget{}
	device.TargetId = addr
	device.LinksId = common2.NetworkDevice_Links_ID
	device.Hosts = make(map[string]*l8tpollaris.L8PHost)
	host := &l8tpollaris.L8PHost{}
	host.HostId = device.TargetId

	host.Configs = make(map[int32]*l8tpollaris.L8PHostProtocol)
	device.Hosts[device.TargetId] = host

	graphQlConfig := &l8tpollaris.L8PHostProtocol{}
	graphQlConfig.Port = int32(port)
	graphQlConfig.Addr = addr
	graphQlConfig.CredId = "sim"
	graphQlConfig.Protocol = l8tpollaris.L8PProtocol_L8PGraphQL
	graphQlConfig.Timeout = 30

	graphQlConfig.Ainfo = &l8tpollaris.AuthInfo{
		NeedAuth: false,
		IsApiKey: true,
		ApiUser:  user,
		ApiKey:   pass,
	}

	host.Configs[int32(graphQlConfig.Protocol)] = graphQlConfig

	return device
}

// SetPolls configures poll cadences for testing and adds all pollaris models
// to the service level agreement. It reduces poll cadences to 3 seconds
// for faster test execution and includes Kubernetes boot polls.
//
// Parameters:
//   - sla: The service level agreement to configure with poll data
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
