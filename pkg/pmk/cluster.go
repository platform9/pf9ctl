// Copyright © 2020 The Platform9 Systems Inc.

package pmk

import (
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/platform9/pf9ctl/pkg/qbert"
	"github.com/platform9/pf9ctl/pkg/ssh"
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
	output, err := c.Executor.RunWithStdout("bash", "-c", cmd)
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

func CreateHeadlessCluster(pf9KubePath string, configTarPath string,
	masterNodeList []string, workerNodeList []string, username string,
	privKeyPath string, password string) error {

	privKey, err := ioutil.ReadFile(privKeyPath)
	if err != nil {
		return fmt.Errorf("error reading key file %s %s", privKeyPath, err)
	}

	err = bootstrapMasters(masterNodeList, username, privKey, password,
		pf9KubePath, configTarPath)
	if err != nil {
		return fmt.Errorf("failed to boostrap masters: %s", err)
	}

	err = bootstrapWorkers(workerNodeList, username, privKey, password,
		pf9KubePath, configTarPath)
	if err != nil {
		return fmt.Errorf("failed to boostrap workers: %s", err)
	}

	return nil
}

func bootstrapMasters(masterNodeList []string, username string, privKey []byte,
	password string, pf9KubePath string, configTarPath string) error {
	for _, masterNode := range masterNodeList {
		err := uploadPackages(masterNode, username, privKey, password,
			pf9KubePath, configTarPath)
		if err != nil {
			return fmt.Errorf("failed to upload packages to %s: %s",
				masterNode, err)
		}
	}
	return nil
}

func bootstrapWorkers(workerNodeList []string, username string, privKey []byte,
	password string, pf9KubePath string, configTarPath string) error {

	for _, workerNode := range workerNodeList {
		// TODO - parallelize this
		err := uploadPackages(workerNode, username, privKey, password,
			pf9KubePath, configTarPath)
		if err != nil {
			return fmt.Errorf("failed to upload packages to %s: %s",
				workerNode, err)
		}
	}
	return nil
}

func uploadPackages(node string, username string, privKey []byte,
	password string, pf9KubePath string, configTarPath string) error {
	client, err := ssh.NewClient(node, 22, username, privKey, password)
	if err != nil {
		return fmt.Errorf("failed to create ssh client to %s: %s", node, err)
	}

	err = client.UploadFile(pf9KubePath, "/tmp/pf9kube.rpm", 0644, nil)
	if err != nil {
		return fmt.Errorf("failed to upload pf9kube RPM to %s: %s", node, err)
	}

	err = client.UploadFile(configTarPath, "/tmp/pf9config.tar", 0644, nil)
	if err != nil {
		return fmt.Errorf("failed to upload config tar to %s: %s", node, err)
	}

	return nil
}
