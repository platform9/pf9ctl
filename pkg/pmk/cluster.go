// Copyright © 2020 The Platform9 Systems Inc.

package pmk

import (
	"fmt"
	"strings"
	"time"

	"github.com/platform9/pf9ctl/pkg/constants"
	"github.com/platform9/pf9ctl/pkg/log"
	"github.com/platform9/pf9ctl/pkg/qbert"
	"github.com/platform9/pf9ctl/pkg/util"
)

// Bootstrap simply preps the local node and attach it as master to a newly
// created cluster.
func Bootstrap(ctx Context, c Client, req qbert.ClusterCreateRequest) error {
	log.Debug("Received a call to boostrap the local node")

	resp, err := util.AskBool("Prep local node for kubernetes cluster")
	if err != nil || !resp {
		log.Errorf("Couldn't fetch user content")
	}

	if err := PrepNode(ctx, c, "", "", "", []string{}); err != nil {
		return fmt.Errorf("Unable to prepnode: %w", err)
	}

	keystoneAuth, err := c.Keystone.GetAuth(
		ctx.Username,
		ctx.Password,
		ctx.Tenant,
	)
	if err != nil {
		log.Fatalf("keystone authentication failed: %s", err.Error())
	}

	log.Info("Creating the cluster...")
	clusterID, err := c.Qbert.CreateCluster(
		req,
		keystoneAuth.ProjectID,
		keystoneAuth.Token)

	if err != nil {
		return fmt.Errorf("Unable to create cluster: %w", err)
	}

	cmd := `\"cat /etc/pf9/host_id.conf | grep ^host_id | cut -d = -f2 | cut -d ' ' -f2\"`
	output, err := c.Executor.RunWithStdout("bash", "-c", cmd)
	if err != nil {
		return fmt.Errorf("Unable to execute command: %w", err)
	}
	nodeID := strings.TrimSuffix(string(output), "\n")

	time.Sleep(constants.WaitPeriod * time.Second)

	log.Info("Attaching node to the cluster...")
	err = c.Qbert.AttachNode(
		clusterID,
		nodeID,
		keystoneAuth.ProjectID, keystoneAuth.Token)

	if err != nil {
		return fmt.Errorf("Unable to attach node: %w", err)
	}

	log.Info("Bootstrap successfully finished")
	return nil
}
