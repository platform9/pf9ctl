package pmk

import (
	"fmt"
	"strings"

	"github.com/platform9/pf9ctl/pkg/log"
	"github.com/platform9/pf9ctl/pkg/pmk/clients"
	"github.com/platform9/pf9ctl/pkg/util"
)

// Bootstrap simply preps the local node and attach it as master to a newly
// created cluster.
func Bootstrap(
	ctx Context,
	c clients.Client,
	req clients.ClusterCreateRequest) error {
	log.Info("Received a call to boostrap the local node")

	prep, err := util.AskBool("PrepLocal node for kubernetes cluster")
	if err != nil {
		return fmt.Errorf("Unable to capture user response: %w", err)
	}

	if prep {
		err = PrepNode(ctx, c, "", "", "", []string{})
		if err != nil {
			return fmt.Errorf("Unable to prepnode: %w", err)
		}
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
	log.Info("Cluster created successfully")

	cmd := `cat /etc/pf9/host_id.conf | grep ^host_id | cut -d = -f2 | cut -d ' ' -f2`
	output, err := c.Executor.RunWithStdout("bash", "-c", cmd)
	if err != nil {
		return fmt.Errorf("Unable to execute command: %w", err)
	}
	log.Info("Attaching node to the cluster...")
	nodeID := strings.TrimSuffix(output, "\n")
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
