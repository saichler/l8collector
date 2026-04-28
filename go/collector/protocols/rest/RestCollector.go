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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8srlz/go/serialize/object"
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
	csrfToken    string
	csrfRegex    *regexp.Regexp
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

	ainfo := hostConn.Ainfo
	if ainfo.SessionAuth {
		jar, _ := cookiejar.New(nil)
		if len(ainfo.PresetCookies) > 0 {
			u, _ := url.Parse(scheme + "://" + hostConn.Addr)
			cookies := make([]*http.Cookie, 0, len(ainfo.PresetCookies))
			for name, value := range ainfo.PresetCookies {
				cookies = append(cookies, &http.Cookie{Name: name, Value: value, Path: "/"})
			}
			jar.SetCookies(u, cookies)
		}
		this.httpClient = &http.Client{Jar: jar}
		if ainfo.CsrfPattern != "" {
			this.csrfRegex = regexp.MustCompile(ainfo.CsrfPattern)
		}
	} else {
		this.httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	}

	return nil
}

// Protocol returns the protocol type identifier for REST.
func (this *RestCollector) Protocol() l8tpollaris.L8PProtocol {
	return l8tpollaris.L8PProtocol_L8PRESTAPI
}

// parseWhat parses the poll.What field to extract HTTP method, endpoint, body, and content type.
// Format: "METHOD::endpoint::body" or "METHOD::endpoint::body::content-type"
func (this *RestCollector) parseWhat(poll *l8tpollaris.L8Poll) (string, string, string, string, error) {
	tokens := strings.Split(poll.What, "::")
	if len(tokens) < 3 {
		return "", "", "", "", fmt.Errorf("invalid What format")
	}

	switch tokens[0] {
	case "GET", "POST", "PUT", "PATCH", "DELETE":
	default:
		return "", "", "", "", fmt.Errorf("invalid What method: %s", tokens[0])
	}

	contentType := "application/json"
	if len(tokens) >= 4 && tokens[3] != "" {
		contentType = tokens[3]
	}

	return tokens[0], tokens[1], tokens[2], contentType, nil
}

// Connect performs the authentication handshake.
// For session-based auth, it logs in and extracts CSRF tokens.
// For non-session auth, it is a no-op.
func (this *RestCollector) Connect() error {
	ainfo := this.hostProtocol.Ainfo
	if !ainfo.SessionAuth {
		this.connected = true
		return nil
	}

	err := this.sessionLogin()
	if err != nil {
		return err
	}
	this.connected = true
	return nil
}

// sessionLogin posts the login payload and establishes the session.
func (this *RestCollector) sessionLogin() error {
	ainfo := this.hostProtocol.Ainfo
	baseScheme := "https://" + this.hostProtocol.Addr

	// Build login body by substituting credentials
	loginBody := ainfo.AuthBody
	loginBody = strings.ReplaceAll(loginBody, "{{user}}", ainfo.ApiUser)
	loginBody = strings.ReplaceAll(loginBody, "{{pass}}", ainfo.ApiKey)

	loginURL := baseScheme + ainfo.AuthPath
	req, err := http.NewRequest("POST", loginURL, strings.NewReader(loginBody))
	if err != nil {
		return errors.New("session login request build failed: " + err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", baseScheme)
	req.Header.Set("Referer", baseScheme+"/")

	resp, err := this.httpClient.Do(req)
	if err != nil {
		return errors.New("session login request failed: " + err.Error())
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.New("session login read failed: " + err.Error())
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New("session login returned status " + resp.Status)
	}

	// Validate success in response
	if ainfo.AuthResp != "" {
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return errors.New("session login parse failed: " + err.Error())
		}
		if val, ok := result[ainfo.AuthResp]; !ok || val == false {
			return errors.New("session login auth response check failed")
		}
	}

	// Visit session page to establish full session context
	if ainfo.SessionPage != "" {
		sessionURL := baseScheme + ainfo.SessionPage
		sResp, err := this.httpClient.Get(sessionURL)
		if err != nil {
			return errors.New("session page fetch failed: " + err.Error())
		}
		defer sResp.Body.Close()
		sBody, _ := io.ReadAll(sResp.Body)
		// Extract CSRF from session page if regex is configured
		if this.csrfRegex != nil {
			matches := this.csrfRegex.FindSubmatch(sBody)
			if len(matches) >= 2 {
				this.csrfToken = string(matches[1])
			}
		}
	}

	// If CSRF source is different from session page, fetch it too
	if ainfo.CsrfSource != "" && ainfo.CsrfSource != ainfo.SessionPage {
		this.refreshCSRF()
	}

	return nil
}

// EnsureSession checks if the session is still valid. If not, re-logs in.
func (this *RestCollector) EnsureSession() error {
	ainfo := this.hostProtocol.Ainfo
	if !ainfo.SessionAuth {
		return nil
	}

	baseScheme := "https://" + this.hostProtocol.Addr
	checkURL := baseScheme + ainfo.SessionPage
	if checkURL == baseScheme {
		checkURL = baseScheme + "/"
	}

	resp, err := this.httpClient.Get(checkURL)
	if err != nil {
		return this.sessionLogin()
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body)

	// A redirect away from the session page means session expired
	if resp.Request != nil && resp.Request.URL != nil {
		reqPath := resp.Request.URL.Path
		sessionPath := ainfo.SessionPage
		if sessionPath != "" && !strings.HasPrefix(reqPath, sessionPath) {
			return this.sessionLogin()
		}
	}
	return nil
}

// refreshCSRF fetches the CSRF source page and extracts the token.
func (this *RestCollector) refreshCSRF() {
	ainfo := this.hostProtocol.Ainfo
	if ainfo.CsrfSource == "" || this.csrfRegex == nil {
		return
	}
	baseScheme := "https://" + this.hostProtocol.Addr
	resp, err := this.httpClient.Get(baseScheme + ainfo.CsrfSource)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	matches := this.csrfRegex.FindSubmatch(body)
	if len(matches) >= 2 {
		this.csrfToken = string(matches[1])
	}
}

// Exec executes a REST API job and stores the raw JSON response in job.Result.
func (this *RestCollector) Exec(job *l8tpollaris.CJob) {
	ainfo := this.hostProtocol.Ainfo

	// Session keepalive before each request
	if ainfo.SessionAuth {
		if err := this.EnsureSession(); err != nil {
			job.ErrorCount++
			job.Error = "session keepalive failed: " + err.Error()
			return
		}
	}

	if !this.connected {
		this.connected = true
	}
	poll, err := pollaris.Poll(job.PollarisName, job.JobName, this.resources)
	if err != nil {
		job.ErrorCount++
		job.Error = err.Error()
		return
	}
	method, endpoint, body, contentType, err := this.parseWhat(poll)
	if err != nil {
		job.ErrorCount++
		job.Error = err.Error()
		return
	}

	fullURL := this.baseURL + endpoint
	var reqBody io.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, fullURL, reqBody)
	if err != nil {
		job.ErrorCount++
		job.Error = err.Error()
		return
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")

	// Inject CSRF token if present
	if this.csrfToken != "" {
		req.Header.Set("X-CSRF-Token", this.csrfToken)
	}

	// For AJAX-style requests, add standard headers
	if ainfo.SessionAuth {
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		baseScheme := "https://" + this.hostProtocol.Addr
		req.Header.Set("Origin", baseScheme)
	}

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

	// Wrap JSON in a CMap so the parser can deserialize it
	cmap := &l8tpollaris.CMap{}
	cmap.Data = make(map[string][]byte)
	enc := object.NewEncode()
	enc.Add(string(jsonBytes))
	cmap.Data["json"] = enc.Data()

	encMap := object.NewEncode()
	encMap.Add(cmap)
	job.Result = encMap.Data()
}

// Disconnect releases all resources.
func (this *RestCollector) Disconnect() error {
	this.httpClient = nil
	this.hostProtocol = nil
	this.resources = nil
	this.connected = false
	this.csrfToken = ""
	this.csrfRegex = nil
	return nil
}

// Online returns the connection status.
func (this *RestCollector) Online() bool {
	return true
}
