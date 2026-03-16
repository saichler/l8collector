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

// Package rest provides a REST protocol collector implementation for
// the L8Collector service. It enables data collection from REST APIs
// and forwards raw JSON responses to the parser.
package rest

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8types/go/ifs"
)

// RestCollector implements the ProtocolCollector interface for REST APIs.
// It executes HTTP requests and stores raw JSON responses in the job result.
type RestCollector struct {
	httpClient   *http.Client
	hostProtocol *l8tpollaris.L8PHostProtocol
	resources    ifs.IResources
	connected    bool
	baseURL      string
}

// Init initializes the REST collector with the provided host configuration.
func (this *RestCollector) Init(hostConn *l8tpollaris.L8PHostProtocol, r ifs.IResources) error {
	if hostConn.Ainfo == nil {
		return errors.New("host rest auth info connection info is nil")
	}
	this.hostProtocol = hostConn
	this.resources = r

	scheme := "https"
	this.baseURL = scheme + "://" + hostConn.Addr + ":" + strconv.Itoa(int(hostConn.Port))
	if hostConn.HttpPrefix != "" {
		this.baseURL += hostConn.HttpPrefix
	}

	this.httpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	return nil
}

// Protocol returns the protocol type identifier for REST.
func (this *RestCollector) Protocol() l8tpollaris.L8PProtocol {
	return l8tpollaris.L8PProtocol_L8PRESTAPI
}

// parseWhat parses the poll.What field to extract HTTP method, endpoint, and body.
// Format: "METHOD::endpoint::body_json"
func (this *RestCollector) parseWhat(poll *l8tpollaris.L8Poll) (string, string, string, error) {
	tokens := strings.Split(poll.What, "::")
	if len(tokens) != 3 {
		return "", "", "", fmt.Errorf("invalid What format")
	}

	switch tokens[0] {
	case "GET", "POST", "PUT", "PATCH", "DELETE":
	default:
		return "", "", "", fmt.Errorf("invalid What method: %s", tokens[0])
	}

	return tokens[0], tokens[1], tokens[2], nil
}

// Exec executes a REST API job and stores the raw JSON response in job.Result.
func (this *RestCollector) Exec(job *l8tpollaris.CJob) {
	if !this.connected {
		this.connected = true
	}
	poll, err := pollaris.Poll(job.PollarisName, job.JobName, this.resources)
	if err != nil {
		job.ErrorCount++
		job.Error = err.Error()
		return
	}
	method, endpoint, body, err := this.parseWhat(poll)
	if err != nil {
		job.ErrorCount++
		job.Error = err.Error()
		return
	}

	url := this.baseURL + endpoint
	var reqBody io.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		job.ErrorCount++
		job.Error = err.Error()
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := this.httpClient.Do(req)
	if err != nil {
		job.ErrorCount++
		job.Error = err.Error()
		return
	}
	defer resp.Body.Close()

	jsonBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		job.ErrorCount++
		job.Error = err.Error()
		return
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		job.ErrorCount++
		job.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(jsonBytes))
		return
	}

	job.ErrorCount = 0
	job.Result = jsonBytes
}

// Connect is a no-op for the JSON-based REST collector.
func (this *RestCollector) Connect() error {
	this.connected = true
	return nil
}

// Disconnect releases all resources.
func (this *RestCollector) Disconnect() error {
	this.httpClient = nil
	this.hostProtocol = nil
	this.resources = nil
	this.connected = false
	return nil
}

// Online returns the connection status.
func (this *RestCollector) Online() bool {
	return true
}
