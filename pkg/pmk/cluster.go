package pmk

import (
	"fmt"
	"strings"
	"time"

	"github.com/platform9/pf9ctl/pkg/log"
	"github.com/platform9/pf9ctl/pkg/pmk/clients"
	"github.com/platform9/pf9ctl/pkg/util"
)

// Bootstrap simply preps the local node and attach it as master to a newly
// created cluster.
func Bootstrap(ctx Context, c clients.Client, req clients.ClusterCreateRequest) error {
	log.Info.Println("Received a call to boostrap the local node")

	resp, err := util.AskBool("PrepLocal node for kubernetes cluster")
	if err != nil || !resp {
		log.Error.Fatalf("Couldn't fetch user content")
	}

	err = PrepNode(ctx, c, "", "", "", []string{})
	if err != nil {
		return fmt.Errorf("Unable to prepnode: %w", err)
	}

	keystoneAuth, err := c.Keystone.GetAuth(
		ctx.Username,
		ctx.Password,
		ctx.Tenant)

	if err != nil {
		log.Error.Fatalf("keystone authentication failed: %s", err.Error())
	}

	clusterID, err := c.Qbert.CreateCluster(
		req,
		keystoneAuth.ProjectID,
		keystoneAuth.Token)

	if err != nil {
		return fmt.Errorf("Unable to create cluster: %w", err)
	}

	cmd := `cat /etc/pf9/host_id.conf | grep ^host_id | cut -d = -f2 | cut -d ' ' -f2`
	output, err := c.Executor.RunWithStdout("bash", "-c", cmd)
	if err != nil {
		return fmt.Errorf("Unable to execute command: %w", err)
	}
	nodeID := strings.TrimSuffix(string(output), "\n")

	log.Info.Println("Waiting for the cluster to get created")
	time.Sleep(WaitPeriod * time.Second)

	log.Info.Println("Cluster created successfully")
	err = c.Qbert.AttachNode(
		clusterID,
		nodeID,
		keystoneAuth.ProjectID, keystoneAuth.Token)

	if err != nil {
		return fmt.Errorf("Unable to attach node: %w", err)
	}

	log.Info.Printf("\nBootstrap successfully Finished\n")
	return nil
}
