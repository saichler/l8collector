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

// Package graphql provides a GraphQL protocol collector implementation for
// the L8Collector service. It enables data collection from GraphQL APIs
// with support for both API key and token-based authentication.
package graphql

import (
	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8web/go/web/gclient"
	"google.golang.org/protobuf/proto"
)

// GraphQlCollector implements the ProtocolCollector interface for GraphQL APIs.
// It provides the ability to execute GraphQL queries against remote endpoints
// and collect the responses as protobuf-serialized data.
//
// Features:
//   - HTTPS connections with certificate support
//   - API key authentication (X-API-KEY header)
//   - Token-based authentication via login endpoints
//   - Flexible query structure with typed responses
//   - Automatic connection management
//
// The collector uses the l8web/gclient package for GraphQL client operations.
type GraphQlCollector struct {
	client       *gclient.GraphQLClient        // GraphQL client for query execution
	hostProtocol *l8tpollaris.L8PHostProtocol  // Host configuration with connection details
	resources    ifs.IResources                // Layer8 resources for logging and registry
	connected    bool                          // Connection state flag
}

// Init initializes the GraphQL collector with the provided host configuration.
// It creates a GraphQL client configured with the host's address, port,
// authentication settings, and optional TLS certificate.
//
// The client is configured for HTTPS by default. Authentication can be
// either API key-based (using X-API-KEY headers) or token-based (using
// a login endpoint that returns a bearer token).
//
// Parameters:
//   - hostConn: Host protocol configuration containing address, port, and auth info
//   - r: Layer8 resources for accessing security credentials and logging
//
// Returns:
//   - error if client creation fails, nil on success
func (this *GraphQlCollector) Init(hostConn *l8tpollaris.L8PHostProtocol, r ifs.IResources) error {
	clientConfig := &gclient.GraphQLClientConfig{
		Host:          hostConn.Addr,
		Port:          int(hostConn.Port),
		Https:         true,
		TokenRequired: false,
		CertFileName:  hostConn.Cert,
		Prefix:        hostConn.HttpPrefix,
		AuthInfo: &gclient.GraphQLAuthInfo{
			NeedAuth:   hostConn.Ainfo.NeedAuth,
			BodyType:   hostConn.Ainfo.AuthBody,
			UserField:  hostConn.Ainfo.AuthUserField,
			PassField:  hostConn.Ainfo.AuthPassField,
			RespType:   hostConn.Ainfo.AuthResp,
			TokenField: hostConn.Ainfo.AuthToken,
			AuthPath:   hostConn.Ainfo.AuthPath,
			ApiKey:     hostConn.Ainfo.ApiKey,
			ApiUser:    hostConn.Ainfo.ApiUser,
			IsAPIKey:   hostConn.Ainfo.IsApiKey,
		},
	}
	client, err := gclient.NewGraphQLClient(clientConfig, r)
	if err != nil {
		return err
	}
	this.hostProtocol = hostConn
	this.client = client
	this.resources = r
	return nil
}

// Protocol returns the protocol type identifier for GraphQL.
// This is used by the collector service to route jobs to the correct collector.
func (this *GraphQlCollector) Protocol() l8tpollaris.L8PProtocol {
	return l8tpollaris.L8PProtocol_L8PGraphQL
}

// Exec executes a GraphQL query job against the configured endpoint.
// The query is obtained from the pollaris configuration using the job's
// PollarisName and JobName. The response is serialized using protobuf
// and stored in the job's Result field.
//
// The method automatically establishes a connection if not already connected.
// Errors are recorded in the job's Error field and ErrorCount is incremented.
//
// The poll configuration should contain:
//   - What: The GraphQL query string
//   - RespName: The expected response type name for protobuf unmarshaling
//
// Parameters:
//   - job: The collection job containing pollaris reference and result storage
func (this *GraphQlCollector) Exec(job *l8tpollaris.CJob) {
	if !this.connected {
		err := this.Connect()
		if err != nil {
			job.ErrorCount++
			job.Error = err.Error()
			return
		}
	}
	poll, err := pollaris.Poll(job.PollarisName, job.JobName, this.resources)
	if err != nil {
		job.ErrorCount++
		job.Error = err.Error()
		return
	}

	resp, err := this.client.Query(poll.What, nil, poll.RespName, "")
	if err != nil {
		job.ErrorCount++
		job.Error = err.Error()
		return
	}

	job.ErrorCount = 0
	job.Result, _ = proto.Marshal(resp)
}

// Connect establishes the authenticated connection to the GraphQL endpoint.
// It retrieves credentials from the security service using the configured
// credential ID and performs authentication using the GraphQL client.
//
// For API key authentication, the credentials are sent as headers.
// For token-based authentication, a login request is made to obtain a bearer token.
//
// Returns:
//   - error if authentication fails, nil on success
func (this *GraphQlCollector) Connect() error {
	_, username, password, _, err := this.resources.Security().Credential(this.hostProtocol.CredId, "graph", this.resources)
	if err != nil {
		panic(err)
	}
	return this.client.Auth(username, password)
}

// Disconnect closes the GraphQL client connection and releases all resources.
// After calling Disconnect, the collector must be re-initialized before use.
//
// Returns:
//   - Always returns nil (connection cleanup is best-effort)
func (this *GraphQlCollector) Disconnect() error {
	if this.client != nil {
		this.client = nil
	}
	this.hostProtocol = nil
	this.resources = nil
	this.connected = false
	return nil
}

// Online returns the connection status of the GraphQL collector.
// For GraphQL, this always returns true as connections are stateless HTTP requests.
// The actual connectivity is verified during each query execution.
func (this *GraphQlCollector) Online() bool {
	return true
}
