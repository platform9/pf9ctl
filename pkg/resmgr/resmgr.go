// Copyright © 2020 The Platform9 Systems Inc.
package resmgr

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"crypto/tls"
	"net/http"
	"time"

	rhttp "github.com/hashicorp/go-retryablehttp"
	"github.com/platform9/pf9ctl/pkg/util"
	"go.uber.org/zap"
)

type Resmgr interface {
	AuthorizeHost(hostID, token string, version string) error
	GetHostId(token string, hostIP []string) []string
	HostStatus(token string, hostID string) bool
}

type ResmgrImpl struct {
	fqdn          string
	minWait       time.Duration
	maxWait       time.Duration
	maxHttpRetry  int
	allowInsecure bool
}

type hostInfo []struct {
	Extensions struct {
		IPAddress struct {
			Data []string `json:"data"`
		} `json:"ip_address,omitempty"`
	} `json:"extensions,omitempty"`
	ID string `json:"id,omitempty"`
}

func NewResmgr(fqdn string, maxHttpRetry int, minWait, maxWait time.Duration, allowInsecure bool) Resmgr {

	return &ResmgrImpl{fqdn, minWait, maxWait, maxHttpRetry, allowInsecure}
}

// AuthorizeHost registers the host with hostID to the resmgr.
func (c *ResmgrImpl) AuthorizeHost(hostID string, token string, version string) error {
	zap.S().Debugf("Authorizing the host: %s with DU: %s", hostID, c.fqdn)

	client := rhttp.NewClient()
	client.HTTPClient.Transport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	client.RetryWaitMin = c.minWait
	client.RetryWaitMax = c.maxWait
	client.RetryMax = c.maxHttpRetry
	client.CheckRetry = rhttp.CheckRetry(util.RetryPolicyOn404)
	client.Logger = &util.ZapWrapper{}

	url := fmt.Sprintf("%s/resmgr/v1/hosts/%s/roles/pf9-kube", c.fqdn, hostID)
	if len(version) != 0 {
		url = fmt.Sprintf("%s/resmgr/v1/hosts/%s/roles/pf9-kube/versions/%s", c.fqdn, hostID, version)
	}
	req, err := rhttp.NewRequest("PUT", url, nil)
	if err != nil {
		return fmt.Errorf("Unable to create a new request: %w", err)
	}

	req.Header.Set("X-Auth-Token", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Client is unable to send the request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("Unable to authorize host, code: %d", resp.StatusCode)
	}

	return nil
}

func (c *ResmgrImpl) GetHostId(token string, hostIPs []string) []string {
	url := fmt.Sprintf("%s/resmgr/v1/hosts", c.fqdn)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		zap.S().Infof("Unable to create a new request: %w", err)
	}
	req.Header.Set("X-Auth-Token", token)
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		zap.S().Infof("Client is unable to send the request: %w", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		zap.S().Infof("Unable to read resp body: %w", err)
	}

	nodeData := hostInfo{}
	err = json.Unmarshal(body, &nodeData)
	if err != nil {
		zap.S().Debugf("Unable to unmarshal resp body to struct: %w", err)
	}
	var hostUUIDs []string

	for _, hostip := range hostIPs {
		hostNotFound := true
		for _, node := range nodeData {
			for _, ip := range node.Extensions.IPAddress.Data {
				if ip == hostip {
					hostUUIDs = append(hostUUIDs, node.ID)
					hostNotFound = false
				}
			}
		}
		if hostNotFound {
			zap.S().Infof("Unable to find host with IP %v please try again or run prep-node first", hostip)
		}
	}

	return hostUUIDs
}

func (c *ResmgrImpl) HostStatus(token string, hostID string) bool {
	url := fmt.Sprintf("%s/resmgr/v1/hosts/%s", c.fqdn, hostID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		zap.S().Infof("Unable to create a new request: %w", err)
	}
	req.Header.Set("X-Auth-Token", token)
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		zap.S().Infof("Client is unable to send the request: %w", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		zap.S().Infof("Unable to read resp body: %w", err)
	}

	type hostInfo struct {
		Info struct {
			Responding bool `json:"responding"`
		} `json:"info"`
	}
	host := hostInfo{}
	err = json.Unmarshal(body, &host)
	if err != nil {
		zap.S().Debugf("Unable to unmarshal resp body to struct: %w", err)
	}
	return host.Info.Responding
}
