// Copyright © 2020 The Platform9 Systems Inc.

package pmk

import (
	"fmt"
	"strings"
	"time"

	"github.com/platform9/pf9ctl/pkg/client"
	"github.com/platform9/pf9ctl/pkg/keystone"
	"github.com/platform9/pf9ctl/pkg/objects"
	"github.com/platform9/pf9ctl/pkg/qbert"
	"github.com/platform9/pf9ctl/pkg/util"
	"go.uber.org/zap"
)

// Bootstrap simply preps the local node and attach it as master to a newly
// created cluster.
func Bootstrap(ctx objects.Config, c client.Client, req qbert.ClusterCreateRequest, keystoneAuth keystone.KeystoneAuth) error {
	zap.S().Debug("Received a call to boostrap the local node")

	resp, err := util.AskBool("Prep local node for kubernetes cluster")
	if err != nil || !resp {
		zap.S().Errorf("Couldn't fetch user content")
	}

	if err := PrepNode(ctx, c, keystoneAuth); err != nil {
		return fmt.Errorf("Unable to prepnode: %w", err)
	}

	zap.S().Info("Creating the cluster...")
	clusterID, err := c.Qbert.CreateCluster(
		req,
		keystoneAuth.ProjectID,
		keystoneAuth.Token)

	if err != nil {
		return fmt.Errorf("Unable to create cluster: %w", err)
	}

	cmd := `grep host_id /etc/pf9/host_id.conf | cut -d '=' -f2`
	output, err := c.Executor.RunWithStdout("bash", "-c", cmd)
	output = strings.TrimSpace(output)
	if err != nil {
		return fmt.Errorf("Unable to execute command: %w", err)
	}
	nodeID := strings.TrimSuffix(string(output), "\n")

	time.Sleep(ctx.WaitPeriod * time.Second)
	var nodeIDs []string
	nodeIDs = append(nodeIDs, nodeID)
	zap.S().Info("Attaching node to the cluster...")
	err = c.Qbert.AttachNode(
		clusterID,
		keystoneAuth.ProjectID, keystoneAuth.Token, nodeIDs, "worker")

	if err != nil {
		return fmt.Errorf("Unable to attach node: %w", err)
	}

	zap.S().Info("Bootstrap successfully finished")
	return nil
}
