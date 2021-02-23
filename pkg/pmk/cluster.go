// Copyright © 2020 The Platform9 Systems Inc.

package pmk

import (
	"fmt"
	"strings"
	"time"

	"github.com/platform9/pf9ctl/pkg/qbert"
	"github.com/platform9/pf9ctl/pkg/util"
	"go.uber.org/zap"
)

// Bootstrap simply preps the local node and attach it as master to a newly
// created cluster.
func Bootstrap(ctx Config, c Client, req qbert.ClusterCreateRequest) error {
	zap.S().Debug("Received a call to boostrap the local node")

	resp, err := util.AskBool("Prep local node for kubernetes cluster")
	if err != nil || !resp {
		zap.S().Errorf("Couldn't fetch user content")
	}

	if err := PrepNode(ctx, c); err != nil {
		return fmt.Errorf("Unable to prepnode: %w", err)
	}

	keystoneAuth, err := c.Keystone.GetAuth(
		ctx.Username,
		ctx.Password,
		ctx.Tenant,
	)
	if err != nil {
		zap.S().Fatalf("keystone authentication failed: %s", err.Error())
	}

	zap.S().Info("Creating the cluster...")
	clusterID, err := c.Qbert.CreateCluster(
		req,
		keystoneAuth.ProjectID,
		keystoneAuth.Token)

	if err != nil {
		return fmt.Errorf("Unable to create cluster: %w", err)
	}

	cmd := `\"cat /etc/pf9/host_id.conf | grep ^host_id | cut -d = -f2 | cut -d ' ' -f2\"`
	output, err := c.ExecutorPair.Sudoer.RunWithStdout("bash", "-c", cmd)
	if err != nil {
		return fmt.Errorf("Unable to execute command: %w", err)
	}
	nodeID := strings.TrimSuffix(string(output), "\n")

	time.Sleep(ctx.WaitPeriod * time.Second)

	zap.S().Info("Attaching node to the cluster...")
	err = c.Qbert.AttachNode(
		clusterID,
		nodeID,
		keystoneAuth.ProjectID, keystoneAuth.Token)

	if err != nil {
		return fmt.Errorf("Unable to attach node: %w", err)
	}

	zap.S().Info("Bootstrap successfully finished")
	return nil
}
