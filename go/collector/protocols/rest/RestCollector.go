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

// Package rest provides a REST/RESTCONF protocol collector implementation for
// the L8Collector service. It enables data collection from REST APIs with
// support for various HTTP methods and token-based authentication.
package rest

import (
	"fmt"
	"strings"

	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8web/go/web/client"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// RestCollector implements the ProtocolCollector interface for REST/RESTCONF APIs.
// It provides the ability to execute HTTP requests against REST endpoints and
// collect the responses as protobuf-serialized data.
//
// Features:
//   - HTTPS connections with certificate support
//   - Token-based authentication via login endpoints
//   - Support for all HTTP methods (GET, POST, PUT, PATCH, DELETE)
//   - Flexible request body and response type handling
//   - Automatic connection and token management
//
// The poll.What field format is: "METHOD::endpoint::body_json"
// Example: "GET::/api/devices::{"query":"filter"}"
type RestCollector struct {
	client       *client.RestClient            // REST client for HTTP operations
	hostProtocol *l8tpollaris.L8PHostProtocol  // Host configuration with connection details
	resources    ifs.IResources                // Layer8 resources for logging and registry
	connected    bool                          // Connection/authentication state flag
}

// Init initializes the REST collector with the provided host configuration.
// It creates a REST client configured with the host's address, port,
// authentication settings, and optional TLS certificate.
//
// The client is configured for HTTPS by default with token-based authentication.
// The authentication flow uses a login endpoint to obtain a bearer token.
//
// Parameters:
//   - hostConn: Host protocol configuration containing address, port, and auth info
//   - r: Layer8 resources for accessing security credentials and logging
//
// Returns:
//   - error if client creation fails, nil on success
func (this *RestCollector) Init(hostConn *l8tpollaris.L8PHostProtocol, r ifs.IResources) error {
	clientConfig := &client.RestClientConfig{
		Host:          hostConn.Addr,
		Port:          int(hostConn.Port),
		Https:         true,
		TokenRequired: true,
		CertFileName:  hostConn.Cert,
		Prefix:        hostConn.HttpPrefix,
		AuthInfo: &client.RestAuthInfo{
			NeedAuth:   hostConn.Ainfo.NeedAuth,
			BodyType:   hostConn.Ainfo.AuthBody,
			UserField:  hostConn.Ainfo.AuthUserField,
			PassField:  hostConn.Ainfo.AuthPassField,
			RespType:   hostConn.Ainfo.AuthResp,
			TokenField: hostConn.Ainfo.AuthToken,
			AuthPath:   hostConn.Ainfo.AuthPath,
		},
	}
	client, err := client.NewRestClient(clientConfig, r)
	if err != nil {
		return err
	}
	this.hostProtocol = hostConn
	this.client = client
	this.resources = r
	return nil
}

// Protocol returns the protocol type identifier for REST/RESTCONF.
// This is used by the collector service to route jobs to the correct collector.
func (this *RestCollector) Protocol() l8tpollaris.L8PProtocol {
	return l8tpollaris.L8PProtocol_L8PRESTCONF
}

// parseWhat parses the poll.What field to extract HTTP method, endpoint, and body.
// The expected format is: "METHOD::endpoint::body_json"
//
// Supported methods: GET, POST, PUT, PATCH, DELETE
//
// The body is unmarshaled from JSON into a protobuf message type specified
// by poll.BodyName in the registry.
//
// Parameters:
//   - poll: The poll configuration containing the What field to parse
//
// Returns:
//   - method: The HTTP method (GET, POST, etc.)
//   - endpoint: The API endpoint path
//   - body: The request body as a protobuf message
//   - error: Any parsing or validation errors
func (this *RestCollector) parseWhat(poll *l8tpollaris.L8Poll) (string, string, proto.Message, error) {
	tokens := strings.Split(poll.What, "::")
	if len(tokens) != 3 {
		return "", "", nil, fmt.Errorf("invalid What format")
	}

	switch tokens[0] {
	case "GET":
		fallthrough
	case "POST":
		fallthrough
	case "PUT":
		fallthrough
	case "PATCH":
		fallthrough
	case "DELETE":
	default:
		return "", "", nil, fmt.Errorf("invalid What method")
	}

	info, err := this.resources.Registry().Info(poll.BodyName)
	if err != nil {
		return "", "", nil, err
	}
	b, _ := info.NewInstance()
	body := b.(proto.Message)

	err = protojson.Unmarshal([]byte(tokens[2]), body)
	if err != nil {
		return "", "", nil, err
	}
	return tokens[0], tokens[1], body, nil
}

// Exec executes a REST API job against the configured endpoint.
// The method, endpoint, and body are obtained from the pollaris configuration
// using the job's PollarisName and JobName. The response is serialized using
// protobuf and stored in the job's Result field.
//
// The method automatically establishes a connection if not already connected.
// Errors are recorded in the job's Error field and ErrorCount is incremented.
//
// Parameters:
//   - job: The collection job containing pollaris reference and result storage
func (this *RestCollector) Exec(job *l8tpollaris.CJob) {
	if !this.connected {
		err := this.Connect()
		if err != nil {
			job.ErrorCount++
			job.Error = err.Error()
			return
		}
	}
	poll, err := pollaris.Poll(job.PollarisName, job.JobName, this.resources)
	method, endpoint, body, err := this.parseWhat(poll)
	if err != nil {
		job.ErrorCount++
		job.Error = err.Error()
		return
	}

	resp, err := this.client.Do(method, endpoint, poll.RespName, "", "", body, 1)
	if err != nil {
		job.ErrorCount++
		job.Error = err.Error()
		return
	}

	job.ErrorCount = 0
	job.Result, _ = proto.Marshal(resp)
}

// Connect establishes the authenticated connection to the REST endpoint.
// It retrieves credentials from the security service using the configured
// credential ID and performs token-based authentication.
//
// Returns:
//   - error if authentication fails, nil on success
func (this *RestCollector) Connect() error {
	_, user, password, _, err := this.resources.Security().Credential(this.hostProtocol.CredId, "rest", this.resources)
	if err != nil {
		panic(err)
	}
	return this.client.Auth(user, password)
}

// Disconnect closes the REST client connection and releases all resources.
// After calling Disconnect, the collector must be re-initialized before use.
//
// Returns:
//   - Always returns nil (connection cleanup is best-effort)
func (this *RestCollector) Disconnect() error {
	if this.client != nil {
		this.client = nil
	}
	this.hostProtocol = nil
	this.resources = nil
	this.connected = false
	return nil
}

// Online returns the connection status of the REST collector.
// For REST, this always returns true as connections are stateless HTTP requests.
// The actual connectivity is verified during each request execution.
func (this *RestCollector) Online() bool {
	return true
}
