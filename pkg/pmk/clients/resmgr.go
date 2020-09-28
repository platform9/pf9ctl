package clients

import (
	"fmt"

	rhttp "github.com/hashicorp/go-retryablehttp"
	"github.com/platform9/pf9ctl/pkg/log"
	"github.com/platform9/pf9ctl/pkg/util"
)

type Resmgr interface {
	AuthorizeHost(hostUUID string) error
}

type ResmgrImpl struct {
	fqdn string
}

func NewResmgr(fqdn string) (Resmgr, error) {
	return &ResmgrImpl{fqdn: fqdn}, nil
}

// AuthorizeHost registers the host with hostID to the resmgr.
func (c *ResmgrImpl) AuthorizeHost(hostID string, token string) error {
	log.Info.Printf("Received a call to authorize host: %s to fqdn: %s\n", hostID, c.fqdn)

	client := rhttp.NewClient()
	client.RetryMax = HTTPMaxRetry
	client.CheckRetry = rhttp.CheckRetry(util.RetryPolicyOn404)

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
