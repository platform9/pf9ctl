package clients

import (
	"fmt"
	"net/http"

	"github.com/platform9/pf9ctl/pkg/log"
)

type Resmgr interface {
	AuthorizeHost(hostID, token string) error
}

type ResmgrImpl struct {
	fqdn   string
	client HTTP
}

func NewResmgr(fqdn string, client HTTP) Resmgr {
	return &ResmgrImpl{fqdn: fqdn, client: client}
}

// AuthorizeHost registers the host with hostID to the resmgr.
func (c *ResmgrImpl) AuthorizeHost(hostID string, token string) error {
	log.Debugf("Authorizing the host: %s with DU: %s", hostID, c.fqdn)

	url := fmt.Sprintf("%s/resmgr/v1/hosts/%s/roles/pf9-kube", c.fqdn, hostID)
	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return fmt.Errorf("Unable to create a new request: %w", err)
	}

	req.Header.Set("X-Auth-Token", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("Unable to send request to the client: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("Unable to authorize host, code: %d", resp.StatusCode)
	}

	return nil
}
