// Copyright © 2020 The Platform9 Systems Inc.

package pmk

import (
	"fmt"
	"strings"
	"time"

	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/qbert"
	"go.uber.org/zap"
)

// Bootstrap simply preps the local node and attach it as master to a newly
// created cluster.
func Bootstrap(ctx Config, c Client, req qbert.ClusterCreateRequest) error {
	keystoneAuth, err := c.Keystone.GetAuth(
		ctx.Username,
		ctx.Password,
		ctx.Tenant,
		ctx.MfaToken,
	)
	if err != nil {
		zap.S().Fatalf("keystone authentication failed %s", err.Error())
	}

	//token := keystoneAuth.Token

	zap.S().Info("Creating the cluster ", req.Name)
	clusterID, err := c.Qbert.CreateCluster(
		req,
		keystoneAuth.ProjectID,
		keystoneAuth.Token)

	if err != nil {
		return fmt.Errorf("Unable to create cluster"+req.Name+": %w", err)
	}

	cmd := `cat /etc/pf9/host_id.conf | grep ^host_id | cut -d = -f2 | cut -d ' ' -f2`
	output, err := c.Executor.RunWithStdout("bash", "-c", cmd)
	if err != nil {
		return fmt.Errorf("Unable to get host-id %w", err)
	}
	nodeID := strings.TrimSuffix(string(output), "\n")

	/*i := 1
	for i <= 3 {
		hostStatus := Host_Status(c.Executor, ctx.Fqdn, token, nodeID)
		if hostStatus == "false" {
			zap.S().Info("Host is Down...Trying again")
		} else {
			break
		}
		i = i + 1
	}

	if i == 4 {
		zap.S().Fatalf("Host is Down.....Exiting from Bootstrap command")
	}*/

	time.Sleep(120 * time.Second)
	var nodeIDs []string
	nodeIDs = append(nodeIDs, nodeID)

	zap.S().Info("Attaching node to the cluster " + req.Name)

	err = c.Qbert.AttachNode(
		clusterID,
		keystoneAuth.ProjectID, keystoneAuth.Token, nodeIDs, "master")

	if err != nil {
		return fmt.Errorf("Unable to attach node to the cluster"+req.Name+": %w", err)
	}

	zap.S().Info("=======Bootstrap successfully finished========")
	return nil

}

func Host_Status(exec cmdexec.Executor, fqdn string, token string, hostID string) string {
	zap.S().Debug("Getting host status")
	tkn := fmt.Sprintf(`"X-Auth-Token: %v"`, token)
	cmd := fmt.Sprintf("curl -sH %v -X GET %v/resmgr/v1/hosts/%v | jq .info.responding ", tkn, fqdn, hostID)
	status, err := exec.RunWithStdout("bash", "-c", cmd)
	if err != nil {
		zap.S().Fatalf("Unable to get host status : ", err)
	}
	status = strings.TrimSpace(strings.Trim(status, "\n\""))
	zap.S().Debug("Host status is : ", status)
	return status
}
