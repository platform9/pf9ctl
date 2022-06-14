package pmk

import (
	"testing"
	"time"

	"github.com/platform9/pf9ctl/pkg/client"
	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/keystone"
	"github.com/platform9/pf9ctl/pkg/qbert"
	"github.com/platform9/pf9ctl/pkg/resmgr"
	"github.com/stretchr/testify/assert"
)

var executor cmdexec.Executor

func TestNewClient(t *testing.T) {
	type Client struct {
		Resmgr   resmgr.Resmgr
		Keystone keystone.Keystone
		Qbert    qbert.Qbert
		Executor cmdexec.Executor
		Segment  client.Segment
	}

	testcases := map[string]struct {
		Client
	}{
		//Mocking fields of each struct of client
		"CheckPass": {
			Client: Client{
				Resmgr:   resmgr.NewResmgr("fqdn", 15, 10*time.Second, 30*time.Second, true), //(fqdn, HTTPMaxRetry, HTTPRetryMinWait, HTTPRetryMaxWait, allowInsecure)
				Keystone: keystone.NewKeystone("fqdn"),                                       //(fqdn)
				Qbert:    qbert.NewQbert("fqdn"),                                             //(fqdn)
				Executor: executor,
				Segment:  client.NewSegment("fqdn", true), //(fqdn, noTracking)
			},
		},
	}
	for testname, tc := range testcases {
		t.Run(testname, func(t *testing.T) {
			client, err := client.NewClient("fqdn", executor, true, true) //(fqdn, executor, allowInsecure, noTracking)

			if err != nil {
				t.Errorf("Error occurred : %s", err)
			}
			assert.Equal(t, tc.Client.Resmgr, client.Resmgr)
			assert.Equal(t, tc.Client.Keystone, client.Keystone)
			assert.Equal(t, tc.Client.Qbert, client.Qbert)
			assert.Equal(t, tc.Client.Executor, client.Executor)
			assert.Equal(t, tc.Client.Segment, client.Segment)
		})
	}
}
