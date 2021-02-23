// Copyright © 2020 The Platform9 Systems Inc.
package pmk

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/keystone"
	"github.com/platform9/pf9ctl/pkg/qbert"
	"github.com/platform9/pf9ctl/pkg/resmgr"
)

const HTTPMaxRetry = 15
const HTTPRetryMinWait = 10 * time.Second
const HTTPRetryMaxWait = 30 * time.Second

// Clients struct encapsulate the collection of
// external services
type Client struct {
	Resmgr       resmgr.Resmgr
	Keystone     keystone.Keystone
	Qbert        qbert.Qbert
	ExecutorPair cmdexec.ExecutorPair
	Segment      Segment
}

// New creates the clients needed by the CLI
// to interact with the external services.
func NewClient(fqdn string, executorPair cmdexec.ExecutorPair, allowInsecure bool, noTracking bool) (Client, error) {
	// Bring the hammer down to make default http allow insecure
	if allowInsecure {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return Client{
		Resmgr:       resmgr.NewResmgr(fqdn, HTTPMaxRetry, HTTPRetryMinWait, HTTPRetryMaxWait, allowInsecure),
		Keystone:     keystone.NewKeystone(fqdn),
		Qbert:        qbert.NewQbert(fqdn),
		ExecutorPair: executorPair,
		Segment:      NewSegment(fqdn, noTracking),
	}, nil
}
