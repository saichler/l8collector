/*
 * Copyright (c) 2025 Sharon Aicler (saichler@gmail.com)
 *
 * Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
 * You may obtain a copy of the License at:
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package gclient provides a GraphQL HTTP client for communicating with GraphQL APIs.
// It supports both HTTP and HTTPS with custom CA certificates, bearer token authentication,
// API key authentication, GZIP compression, and automatic retry on timeout.
//
// Features:
//   - GraphQL query and mutation execution
//   - Variable support for parameterized queries
//   - Automatic GraphQL error parsing and reporting
//   - HTTP/HTTPS with TLS certificate verification
//   - Bearer token and API key authentication
//   - GZIP response decompression
//   - Automatic retry on timeout (up to 5 attempts with 5-second backoff)
//   - Protocol Buffer response mapping via protojson
//
// Example usage:
//
//	config := &GraphQLClientConfig{
//	    Host:     "api.example.com",
//	    Port:     443,
//	    Https:    true,
//	    Endpoint: "/graphql",
//	}
//	client, _ := NewGraphQLClient(config, resources)
//	query := `query { users { id name } }`
//	response, _ := client.Query(query, nil, "UserList", "users")
package gclient

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	nethttp "net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/saichler/l8types/go/ifs"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// GraphQLClient is an HTTP client for communicating with GraphQL APIs.
// It handles authentication, request building, and response parsing with
// Protocol Buffer support.
type GraphQLClient struct {
	GraphQLClientConfig                // Embedded configuration
	httpClient          *nethttp.Client // Underlying HTTP client with TLS config
	resources           ifs.IResources  // Layer 8 resources for type registry access
}

// GraphQLClientConfig contains configuration options for creating a GraphQL client.
type GraphQLClientConfig struct {
	Host          string           // Target server hostname (e.g., "api.example.com")
	Prefix        string           // URL prefix for requests (e.g., "/api/v1")
	Port          int              // Target server port
	Https         bool             // Enable HTTPS connections
	TokenRequired bool             // Require bearer token for requests
	Token         string           // Current bearer token (set by Auth() or manually)
	CertFileName  string           // Path to CA certificate file for TLS verification
	AuthInfo      *GraphQLAuthInfo // Authentication configuration
	Endpoint      string           // GraphQL endpoint path (default: "/graphql")
}

// GraphQLAuthInfo contains authentication configuration for the GraphQL client.
// Supports two modes: bearer token authentication and API key authentication.
type GraphQLAuthInfo struct {
	NeedAuth   bool   // Enable bearer token authentication flow
	BodyType   string // Protocol Buffer type name for auth request body
	UserField  string // Field name for username in auth request
	PassField  string // Field name for password in auth request
	RespType   string // Protocol Buffer type name for auth response
	TokenField string // Field name containing token in auth response
	AuthPath   string // Endpoint path for authentication (e.g., "/auth")
	IsAPIKey   bool   // Use API key authentication instead of bearer token
	ApiUser    string // API user ID (sent as X-USER-ID header)
	ApiKey     string // API key (sent as X-API-KEY header)
}

// GraphQLRequest represents a GraphQL operation request with query and optional variables.
type GraphQLRequest struct {
	Query     string                 `json:"query"`               // GraphQL query or mutation string
	Variables map[string]interface{} `json:"variables,omitempty"` // Optional variables for the query
}

// GraphQLResponse represents the standard GraphQL response structure with data and errors.
type GraphQLResponse struct {
	Data   json.RawMessage `json:"data,omitempty"`   // Query result data
	Errors []GraphQLError  `json:"errors,omitempty"` // GraphQL execution errors
}

// GraphQLError represents a single error from a GraphQL operation.
type GraphQLError struct {
	Message    string                 `json:"message"`              // Error message
	Locations  []GraphQLErrorLocation `json:"locations,omitempty"`  // Source locations where error occurred
	Path       []interface{}          `json:"path,omitempty"`       // Path to the field that caused the error
	Extensions map[string]interface{} `json:"extensions,omitempty"` // Additional error metadata
}

// GraphQLErrorLocation represents the line and column in the query where an error occurred.
type GraphQLErrorLocation struct {
	Line   int `json:"line"`   // Line number (1-indexed)
	Column int `json:"column"` // Column number (1-indexed)
}

// NewGraphQLClient creates a new GraphQL client with the provided configuration.
// For HTTPS connections, it configures TLS:
//   - If CertFileName is provided, it uses that CA certificate for verification
//   - Otherwise, it uses InsecureSkipVerify (suitable for self-signed certs)
//
// If Endpoint is not specified, it defaults to "/graphql".
// Returns an error if the certificate file cannot be read.
func NewGraphQLClient(config *GraphQLClientConfig, resources ifs.IResources) (*GraphQLClient, error) {
	gc := &GraphQLClient{}
	gc.CertFileName = config.CertFileName
	gc.Host = config.Host
	gc.Https = config.Https
	gc.AuthInfo = config.AuthInfo
	gc.Prefix = config.Prefix
	gc.Port = config.Port
	gc.TokenRequired = config.TokenRequired
	gc.Token = config.Token
	gc.resources = resources
	gc.Endpoint = config.Endpoint
	if gc.Endpoint == "" {
		gc.Endpoint = "/graphql"
	}

	if !gc.Https {
		gc.httpClient = &nethttp.Client{}
	} else {
		if gc.CertFileName != "" {
			caCert, err := os.ReadFile(gc.CertFileName)
			if err != nil {
				return nil, err
			}
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)
			gc.httpClient = &nethttp.Client{
				Transport: &nethttp.Transport{
					TLSClientConfig: &tls.Config{
						RootCAs:    caCertPool,
						ClientAuth: tls.NoClientCert,
						ServerName: gc.Host,
					},
				},
			}
		} else {
			gc.httpClient = &nethttp.Client{
				Transport: &nethttp.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
						ServerName:         gc.Host,
					},
				},
			}
		}
	}

	return gc, nil
}

// buildURL constructs the full URL for a GraphQL request, combining the host,
// port, prefix, and endpoint. The prefix is not added for the /auth endpoint.
func (gc *GraphQLClient) buildURL(end string) string {
	url := bytes.Buffer{}
	url.WriteString("http")
	if gc.Https {
		url.WriteString("s")
	}
	url.WriteString("://")
	url.WriteString(gc.Host)
	url.WriteString(":")
	url.WriteString(strconv.Itoa(gc.Port))
	if gc.Prefix != "" && end != "/auth" {
		url.WriteString(gc.Prefix)
	}
	url.WriteString(end)
	fmt.Println("GraphQL Client URL:", url.String())
	return url.String()
}

// request creates an HTTP POST request for a GraphQL operation with proper headers.
// It marshals the GraphQL request to JSON, sets Authorization header if a token
// is available, and adds API key headers if configured.
// Panics if TokenRequired is true but no token is available for non-auth endpoints.
func (gc *GraphQLClient) request(end string, gqlRequest *GraphQLRequest) (*nethttp.Request, error) {
	body, err := json.Marshal(gqlRequest)
	if err != nil {
		return nil, err
	}

	url := gc.buildURL(end)
	request, err := nethttp.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	if gc.TokenRequired && gc.Token == "" && gc.Https && !gc.isAuthPath(end) {
		panic("No token with secure connection!")
	}

	if gc.TokenRequired && gc.Token != "" {
		request.Header.Set("Authorization", "Bearer "+gc.Token)
	}
	request.Header.Add("content-type", "application/json")
	request.Header.Add("Accept", "application/json, text/plain, */*")
	request.Header.Add("Access-Control-Allow-Origin", "*")
	if gc.AuthInfo != nil && gc.AuthInfo.IsAPIKey {
		request.Header.Add("X-USER-ID", gc.AuthInfo.ApiUser)
		request.Header.Add("X-API-KEY", gc.AuthInfo.ApiKey)
	}
	return request, nil
}

// isAuthPath checks if the endpoint is the configured authentication path.
// Used to skip token requirements for the auth endpoint itself.
func (gc *GraphQLClient) isAuthPath(end string) bool {
	if gc.AuthInfo == nil {
		return false
	}
	if strings.HasSuffix(gc.AuthInfo.AuthPath, end) {
		return true
	}
	return false
}

// is200 checks if an HTTP status string represents a successful response (2xx).
// Parses the numeric status code from the status line (e.g., "200 OK").
func is200(status string) (bool, error) {
	index := strings.Index(status, " ")
	stat, err := strconv.Atoi(status[0:index])
	if err != nil {
		return false, err
	}
	if stat >= 200 && stat <= 299 {
		return true, nil
	}
	return false, nil
}

// isTimeout checks if an error indicates a timeout or connection issue.
// If so, it sleeps for 5 seconds before returning true to enable retry.
// Detects: "connection reset by peer", "timeout", "connection timed out".
func isTimeout(err error) bool {
	if strings.Contains(err.Error(), "connection reset by peer") ||
		strings.Contains(err.Error(), "timeout") ||
		strings.Contains(err.Error(), "connection timed out") {
		time.Sleep(time.Second * 5)
		return true
	}
	return false
}

// Auth performs authentication using a GraphQL login mutation.
// It constructs a login mutation based on AuthInfo configuration, executes it,
// and extracts the bearer token from the response. The token is stored in
// gc.Token for use in subsequent requests.
//
// The generated mutation format is:
// mutation { login(input: { user: "...", pass: "..." }) { token } }
//
// Returns nil if NeedAuth is false or if authentication succeeds.
func (gc *GraphQLClient) Auth(user, pass string) error {
	if gc.AuthInfo == nil || !gc.AuthInfo.NeedAuth {
		return nil
	}

	info, err := gc.resources.Registry().Info(gc.AuthInfo.BodyType)
	if err != nil {
		return err
	}

	creds, _ := info.NewInstance()
	credsVal := reflect.ValueOf(creds).Elem()
	if !credsVal.FieldByName(gc.AuthInfo.UserField).CanSet() || !credsVal.FieldByName(gc.AuthInfo.PassField).CanSet() {
		return errors.New("invalid credential field names")
	}

	credsVal.FieldByName(gc.AuthInfo.UserField).Set(reflect.ValueOf(user))
	credsVal.FieldByName(gc.AuthInfo.PassField).Set(reflect.ValueOf(pass))

	// For GraphQL auth, we need to construct a mutation or query
	// This is a simplified version - you may need to customize based on your auth schema
	authQuery := fmt.Sprintf(`mutation { login(input: { %s: "%s", %s: "%s" }) { %s } }`,
		strings.ToLower(gc.AuthInfo.UserField[:1])+gc.AuthInfo.UserField[1:],
		user,
		strings.ToLower(gc.AuthInfo.PassField[:1])+gc.AuthInfo.PassField[1:],
		pass,
		strings.ToLower(gc.AuthInfo.TokenField[:1])+gc.AuthInfo.TokenField[1:])

	token, err := gc.Execute(authQuery, nil, gc.AuthInfo.RespType, gc.AuthInfo.TokenField, 5)
	if err != nil {
		return err
	}

	tokenVal := reflect.ValueOf(token).Elem()
	if !tokenVal.FieldByName(gc.AuthInfo.TokenField).CanSet() {
		return errors.New("invalid token field name")
	}

	t, ok := tokenVal.FieldByName(gc.AuthInfo.TokenField).Interface().(string)
	if !ok {
		return errors.New("invalid token field value, should be string")
	}

	gc.Token = t
	return nil
}

// Execute sends a GraphQL query or mutation and returns the response as a Protocol Buffer.
//
// Parameters:
//   - query: GraphQL query or mutation string
//   - variables: Optional map of variables for parameterized queries
//   - responseType: Protocol Buffer type name for deserializing the response
//   - responseAttribute: Field name to extract from the "data" object (e.g., "users" for data.users)
//   - tryCount: Current retry attempt (starts at 1, max 5)
//
// Handles GZIP response decompression automatically. Parses GraphQL errors and returns
// them as Go errors. Retries on timeout errors up to 5 times with 5-second backoff.
func (gc *GraphQLClient) Execute(query string, variables map[string]interface{}, responseType, responseAttribute string, tryCount int) (proto.Message, error) {
	gqlRequest := &GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	request, err := gc.request(gc.Endpoint, gqlRequest)
	if err != nil {
		return nil, err
	}

	// Execute the request
	response, err := gc.httpClient.Do(request)
	if err != nil {
		if isTimeout(err) {
			if tryCount <= 5 {
				return gc.Execute(query, variables, responseType, responseAttribute, tryCount+1)
			}
		}
		return nil, err
	}

	var jsonBytes []byte

	switch response.Header.Get("Content-Encoding") {
	case "gzip":
		reader, _ := gzip.NewReader(response.Body)
		jsonBytes, _ = io.ReadAll(reader)
		defer reader.Close()
	default:
		jsonBytes, _ = io.ReadAll(response.Body)
	}

	ok, err := is200(response.Status)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("GraphQL request failed with status " + response.Status + ":" + string(jsonBytes))
	}

	// Parse GraphQL response
	var gqlResponse GraphQLResponse
	err = json.Unmarshal(jsonBytes, &gqlResponse)
	if err != nil {
		return nil, err
	}

	// Check for GraphQL errors
	if len(gqlResponse.Errors) > 0 {
		errMsg := "GraphQL errors: "
		for i, gqlErr := range gqlResponse.Errors {
			if i > 0 {
				errMsg += "; "
			}
			errMsg += gqlErr.Message
		}
		return nil, errors.New(errMsg)
	}

	if responseType == "" {
		return nil, nil
	}

	info, err := gc.resources.Registry().Info(responseType)
	if err != nil {
		return nil, err
	}
	_interface, err := info.NewInstance()
	if err != nil {
		return nil, err
	}

	responsePb := _interface.(proto.Message)

	// Extract the data field
	dataBytes := gqlResponse.Data
	if responseAttribute != "" {
		// Extract nested field from data
		var dataMap map[string]json.RawMessage
		err = json.Unmarshal(dataBytes, &dataMap)
		if err != nil {
			return nil, err
		}
		if attrData, ok := dataMap[responseAttribute]; ok {
			dataBytes = attrData
		} else {
			return nil, errors.New("response attribute '" + responseAttribute + "' not found in GraphQL response")
		}
	}

	err = protojson.Unmarshal(dataBytes, responsePb)
	if err != nil {
		fmt.Println(string(dataBytes))
	}
	return responsePb, err
}

// Query executes a GraphQL query and returns the response as a Protocol Buffer.
// Convenience wrapper for Execute() that starts with tryCount=1.
//
// Example:
//
//	query := `query GetUsers($limit: Int!) { users(limit: $limit) { id name } }`
//	vars := map[string]interface{}{"limit": 10}
//	response, _ := client.Query(query, vars, "UserList", "users")
func (gc *GraphQLClient) Query(query string, variables map[string]interface{}, responseType, responseAttribute string) (proto.Message, error) {
	return gc.Execute(query, variables, responseType, responseAttribute, 1)
}

// Mutate executes a GraphQL mutation and returns the response as a Protocol Buffer.
// Convenience wrapper for Execute() that starts with tryCount=1.
// Semantically identical to Query() but named for clarity when performing mutations.
//
// Example:
//
//	mutation := `mutation CreateUser($input: UserInput!) { createUser(input: $input) { id } }`
//	vars := map[string]interface{}{"input": map[string]interface{}{"name": "John"}}
//	response, _ := client.Mutate(mutation, vars, "User", "createUser")
func (gc *GraphQLClient) Mutate(mutation string, variables map[string]interface{}, responseType, responseAttribute string) (proto.Message, error) {
	return gc.Execute(mutation, variables, responseType, responseAttribute, 1)
}
