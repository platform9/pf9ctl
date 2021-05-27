// Copyright © 2020 The Platform9 Systems Inc.
package resmgr

import (
	"fmt"

	"crypto/tls"
	"net/http"
	"time"

	rhttp "github.com/hashicorp/go-retryablehttp"
	"github.com/platform9/pf9ctl/pkg/util"
	"go.uber.org/zap"
)

type Resmgr interface {
	AuthorizeHost(hostID, token string) error
}

type ResmgrImpl struct {
	fqdn          string
	minWait       time.Duration
	maxWait       time.Duration
	maxHttpRetry  int
	allowInsecure bool
}

func NewResmgr(fqdn string, maxHttpRetry int, minWait, maxWait time.Duration, allowInsecure bool) Resmgr {

	return &ResmgrImpl{fqdn, minWait, maxWait, maxHttpRetry, allowInsecure}
}

// AuthorizeHost registers the host with hostID to the resmgr.
func (c *ResmgrImpl) AuthorizeHost(hostID string, token string) error {
	zap.S().Debugf("Authorizing the host: %s with DU: %s", hostID, c.fqdn)

	client := rhttp.NewClient()
	client.HTTPClient.Transport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	client.RetryWaitMin = c.minWait
	client.RetryWaitMax = c.maxWait
	client.RetryMax = c.maxHttpRetry
	client.CheckRetry = rhttp.CheckRetry(util.RetryPolicyOn404)
	client.Logger = &util.ZapWrapper{}

	url := fmt.Sprintf("%s/resmgr/v1/hosts/%s/roles/pf9-kube", c.fqdn, hostID)
	req, err := rhttp.NewRequest("PUT", url, nil)
	if err != nil {
		return fmt.Errorf("Unable to create a new request: %w", err)
	}

	req.Header.Set("X-Auth-Token", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Unable to send request to the client: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("Unable to authorize host, code: %d", resp.StatusCode)
	}

	return nil
}
