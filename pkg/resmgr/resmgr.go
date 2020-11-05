// Copyright © 2020 The Platform9 Systems Inc.
package resmgr

import (
	"fmt"

	rhttp "github.com/hashicorp/go-retryablehttp"
	"go.uber.org/zap"
	"github.com/platform9/pf9ctl/pkg/util"
)

type Resmgr interface {
	AuthorizeHost(hostID, token string) error
}

type ResmgrImpl struct {
	fqdn string
	maxHttpRetry int
}

func NewResmgr(fqdn string, maxHttpRetry int) Resmgr {
	return &ResmgrImpl{fqdn, maxHttpRetry}
}

// AuthorizeHost registers the host with hostID to the resmgr.
func (c *ResmgrImpl) AuthorizeHost(hostID string, token string) error {
	zap.S().Debugf("Authorizing the host: %s with DU: %s", hostID, c.fqdn)

	client := rhttp.NewClient()
	client.RetryMax = c.maxHttpRetry
	client.CheckRetry = rhttp.CheckRetry(util.RetryPolicyOn404)
	client.Logger = nil

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
