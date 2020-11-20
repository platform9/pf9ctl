// Copyright © 2020 The Platform9 Systems Inc.
package pmk

import (
	"github.com/platform9/pf9ctl/pkg/qbert"
	"github.com/platform9/pf9ctl/pkg/keystone"
	"github.com/platform9/pf9ctl/pkg/resmgr"
	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"net/http"
	"crypto/tls"

)

const HTTPMaxRetry = 5

// Clients struct encapsulate the collection of
// external services
type Client struct {
	Resmgr   resmgr.Resmgr
	Keystone keystone.Keystone
	Qbert    qbert.Qbert
	Executor cmdexec.Executor
	Segment  Segment
}

// New creates the clients needed by the CLI
// to interact with the external services.
func NewClient(fqdn string, executor cmdexec.Executor, allowInsecure bool) (Client, error) {
	// Bring the hammer down to make default http allow insecure
	if allowInsecure {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return Client{
		Resmgr:   resmgr.NewResmgr(fqdn, HTTPMaxRetry, allowInsecure),
		Keystone: keystone.NewKeystone(fqdn),
		Qbert:    qbert.NewQbert(fqdn),
		Executor: executor,
		Segment:  NewSegment(fqdn),
	}, nil
}
