package client

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type RestClient struct {
	RestClientConfig
	httpClient *http.Client
	resources  ifs.IResources
}

type RestClientConfig struct {
	Host          string
	Prefix        string
	Port          int32
	Https         bool
	TokenRequired bool
	Token         string
	CertFileName  string
	AuthPaths     []string
}

func NewRestClient(config *RestClientConfig, resources ifs.IResources) (*RestClient, error) {
	rc := &RestClient{}
	rc.CertFileName = config.CertFileName
	rc.Host = config.Host
	rc.Https = config.Https
	rc.AuthPaths = config.AuthPaths
	rc.Prefix = config.Prefix
	rc.Port = config.Port
	rc.TokenRequired = config.TokenRequired
	rc.Token = config.Token
	rc.resources = resources

	if !rc.Https {
		rc.httpClient = &http.Client{}
	} else {
		if rc.CertFileName != "" {
			caCert, err := os.ReadFile(rc.CertFileName)
			if err != nil {
				return nil, err
			}
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)
			rc.httpClient = &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						RootCAs: caCertPool,
						//InsecureSkipVerify: true,
						ClientAuth: tls.NoClientCert,
						ServerName: rc.Host,
					},
				},
			}
		} else {
			rc.httpClient = &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
						ServerName:         rc.Host,
					},
				},
			}
		}

	}

	return rc, nil
}

func (rc *RestClient) buildURL(end, vars string) string {
	url := bytes.Buffer{}
	url.WriteString("http")
	if rc.Https {
		url.WriteString("s")
	}
	url.WriteString("://")
	url.WriteString(rc.Host)
	url.WriteString(":")
	url.WriteString(strconv.Itoa(int(rc.Port)))
	if rc.Prefix != "" && end != "/auth" {
		url.WriteString(rc.Prefix)
	}
	url.WriteString(end)
	url.WriteString(vars)
	fmt.Println("Client URL:", url.String())
	return url.String()
}

func (rc *RestClient) request(method, end, vars string, pbBody proto.Message) (*http.Request, error) {
	var body []byte
	var err error
	if pbBody != nil && vars == "" {
		body, err = protojson.Marshal(pbBody)
		if err != nil {
			return nil, err
		}
	}
	url := rc.buildURL(end, vars)
	request, err := http.NewRequest(method, url, bytes.NewReader([]byte(body)))
	if err != nil {
		return nil, err
	}

	if rc.TokenRequired && rc.Token == "" && rc.Https && !rc.isAuthPath(end) {
		panic("No token with secure connection!")
	}

	if rc.TokenRequired && rc.Token != "" {
		request.Header.Set("Authorization", "Bearer "+rc.Token)
	}
	request.Header.Add("content-type", "application/json")
	request.Header.Add("Accept", "application/json, text/plain, */*")
	request.Header.Add("Access-Control-Allow-Origin", "*")
	return request, nil
}

func (rc *RestClient) isAuthPath(end string) bool {
	if rc.AuthPaths == nil {
		return false
	}
	for _, ap := range rc.AuthPaths {
		if strings.Contains(end, ap) {
			return true
		}
	}
	return false
}

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

func isTimeout(err error) bool {
	if strings.Contains(err.Error(), "connection reset by peer") ||
		strings.Contains(err.Error(), "timeout") ||
		strings.Contains(err.Error(), "connection timed out") {
		time.Sleep(time.Second * 5)
		return true
	}
	return false
}

func (rc *RestClient) Auth(user, pass string) error {
	creds := &l8api.AuthUser{User: user, Pass: pass}
	token, err := rc.Do("POST", "/auth", "AuthToken", "", "", creds, 5)
	if err != nil {
		return err
	}
	rc.Token = token.(*l8api.AuthToken).Token
	return nil
}

func (rc *RestClient) Do(method, end, responseType, responseAttribute, vars string, pbBody proto.Message, tryCount int) (proto.Message, error) {

	request, err := rc.request(method, end, vars, pbBody)
	if err != nil {
		return nil, err
	}

	//Execute the request
	response, err := rc.httpClient.Do(request)
	if err != nil {
		if isTimeout(err) {
			if tryCount <= 5 {
				return rc.Do(method, end, responseType, responseAttribute, vars, pbBody, tryCount+1)
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
		return nil, errors.New(method + " failed with status " + response.Status + ":" + string(jsonBytes))
	}

	if responseType == "" {
		return nil, err
	}

	info, err := rc.resources.Registry().Info(responseType)
	if err != nil {
		return nil, err
	}
	_interface, err := info.NewInstance()
	if err != nil {
		return nil, err
	}

	responsePb := _interface.(proto.Message)
	if responseAttribute != "" {
		buff := bytes.Buffer{}
		buff.WriteString("{\"")
		buff.WriteString(responseAttribute)
		buff.WriteString("\": ")
		buff.Write(jsonBytes)
		buff.WriteString("}")
		jsonBytes = buff.Bytes()
	}
	err = protojson.Unmarshal(jsonBytes, responsePb)
	if err != nil {
		fmt.Println(string(jsonBytes))
	}
	return responsePb, err
}

func (rc *RestClient) GET(end, responseType, responseAttribute, vars string, pbBody proto.Message) (proto.Message, error) {
	return rc.Do("GET", end, responseType, responseAttribute, vars, pbBody, 1)
}

func (rc *RestClient) POST(end, responseType, responseAttribute, vars string, pbBody proto.Message) (proto.Message, error) {
	return rc.Do("POST", end, responseType, responseAttribute, vars, pbBody, 1)
}

func (rc *RestClient) PUT(end, responseType, responseAttribute, vars string, pbBody proto.Message) (proto.Message, error) {
	return rc.Do("PUT", end, responseType, responseAttribute, vars, pbBody, 1)
}

func (rc *RestClient) PATCH(end, responseType, responseAttribute, vars string, pbBody proto.Message) (proto.Message, error) {
	return rc.Do("PATCH", end, responseType, responseAttribute, vars, pbBody, 1)
}

func (rc *RestClient) DELETE(end, responseType, responseAttribute, vars string, pbBody proto.Message) (proto.Message, error) {
	return rc.Do("DELETE", end, responseType, responseAttribute, vars, pbBody, 1)
}
