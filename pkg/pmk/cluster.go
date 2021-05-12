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

type ClusterConf struct {
	// Public variables
	MasterNodeList []string
	WorkerNodeList []string
	Username       string
	Password       string
	PrivKeyPath    string
	Pf9KubePath    string
	ConfigTarPath  string

	// private variables
	privKey []byte
}

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

	cluster := ClusterConf{
		MasterNodeList: masterNodeList,
		WorkerNodeList: workerNodeList,
		Username:       username,
		Password:       password,
		PrivKeyPath:    privKeyPath,
		privKey:        privKey,
		Pf9KubePath:    pf9KubePath,
		ConfigTarPath:  configTarPath,
	}
	err = cluster.BootstrapMasters()
	if err != nil {
		return fmt.Errorf("failed to boostrap masters: %s", err)
	}

	err = cluster.BootstrapWorkers()
	if err != nil {
		return fmt.Errorf("failed to boostrap workers: %s", err)
	}

	return nil
}

func UpgradeHeadlessCluster(pf9KubePath string, configTarPath string,
	masterNodeList []string, workerNodeList []string, username string,
	privKeyPath string, password string) error {

	privKey, err := ioutil.ReadFile(privKeyPath)
	if err != nil {
		return fmt.Errorf("error reading key file %s %s", privKeyPath, err)
	}

	cluster := ClusterConf{
		MasterNodeList: masterNodeList,
		WorkerNodeList: workerNodeList,
		Username:       username,
		Password:       password,
		PrivKeyPath:    privKeyPath,
		privKey:        privKey,
		Pf9KubePath:    pf9KubePath,
		ConfigTarPath:  configTarPath,
	}

	err = cluster.upgradeMasters()
	if err != nil {
		zap.S().Error("Error upgrading master nodes")
		return err
	}

	err = cluster.upgradeWorkers()
	if err != nil {
		zap.S().Error("Error upgrading worker nodes")
		return err
	}
	return nil
}

func (c *ClusterConf) upgradeWorkers() error {
	for _, node := range c.WorkerNodeList {
		err := c.upgradeNode(node)
		if err != nil {
			zap.S().Error("Error upgrading master node: ", node)
			return err
		}
	}
	return nil
}

func (c *ClusterConf) upgradeMasters() error {
	for _, node := range c.MasterNodeList {
		err := c.upgradeNode(node)
		if err != nil {
			zap.S().Error("Error upgrading master node: ", node)
			return err
		}
	}
	return nil
}

func (cluster *ClusterConf) BootstrapMasters() error {
	for _, masterNode := range cluster.MasterNodeList {
		err := cluster.configureNode(masterNode)
		if err != nil {
			return fmt.Errorf("failed to upload packages to %s: %s",
				masterNode, err)
		}
	}
	return nil
}

func (cluster *ClusterConf) BootstrapWorkers() error {

	for _, workerNode := range cluster.WorkerNodeList {
		// TODO - parallelize this
		err := cluster.configureNode(workerNode)
		if err != nil {
			return fmt.Errorf("failed to upload packages to %s: %s",
				workerNode, err)
		}
	}
	return nil
}

func (cluster *ClusterConf) configureNode(node string) error {
	client, err := ssh.NewClient(node, 22, cluster.Username, cluster.privKey,
		cluster.Password)
	if err != nil {
		return fmt.Errorf("failed to create ssh client to %s: %s", node, err)
	}

	err = client.UploadFile(cluster.Pf9KubePath, "/tmp/pf9kube.rpm", 0644, nil)
	if err != nil {
		return fmt.Errorf("failed to upload pf9kube RPM to %s: %s", node, err)
	}

	zap.S().Info("Configuring node: ", node)
	zap.S().Info("Installing pf9-kube package")
	kubeInstallCmd := "yum install -y /tmp/pf9kube.rpm"
	_, _, err = client.RunCommand(kubeInstallCmd)
	if err != nil {
		return fmt.Errorf("failed to install pf9-kube on %s: %s", node, err)
	}

	zap.S().Info("Applying configuration")
	err = client.UploadFile(cluster.ConfigTarPath, "/tmp/pf9config.tgz", 0644, nil)
	if err != nil {
		return fmt.Errorf("failed to upload config tar to %s: %s", node, err)
	}

	_, _, err = client.RunCommand("mkdir -p /etc/pf9")
	if err != nil {
		return fmt.Errorf("failed to create conf dir on %s: %s", node, err)
	}
	_, _, err = client.RunCommand("tar -zxvf /tmp/pf9config.tgz -C /tmp")
	if err != nil {
		return fmt.Errorf("failed to extract config on %s: %s", node, err)
	}

	_, _, err = client.RunCommand("/bin/cp -R -f /tmp/etc/pf9/* /etc/pf9/")
	if err != nil {
		return fmt.Errorf("failed to copy extracted config to correct location on %s: %s", node, err)
	}

	zap.S().Info("Starting nodelet to bring up the cluster on node: ", node)
	_, _, err = client.RunCommand("/opt/pf9/nodelet/nodeletd phases start")
	if err != nil {
		return fmt.Errorf("failed to start nodelet on %s: %s", node, err)
	}

	zap.S().Info("node successfully added to cluster")

	return nil
}

func (c *ClusterConf) upgradeNode(node string) error {

	client, err := ssh.NewClient(node, 22, c.Username, c.privKey,
		c.Password)
	if err != nil {
		return fmt.Errorf("failed to create ssh client to %s: %s", node, err)
	}

	zap.S().Info("Uploading new packages to node: ", node)
	err = client.UploadFile(c.Pf9KubePath, "/tmp/pf9kube.rpm", 0644, nil)
	if err != nil {
		return fmt.Errorf("failed to upload pf9kube RPM to %s: %s", node, err)
	}

	err = client.UploadFile(c.ConfigTarPath, "/tmp/pf9config.tgz", 0644, nil)
	if err != nil {
		return fmt.Errorf("failed to upload config tar to %s: %s", node, err)
	}

	zap.S().Info("Removing old pf9-kube package")
	_, _, err = client.RunCommand("yum erase -y pf9-kube")
	if err != nil {
		return fmt.Errorf("Failed to remove older package from %s: %s", node, err)
	}

	zap.S().Infof("Installing new pf9-kube package")
	_, _, err = client.RunCommand("yum install -y /tmp/pf9kube.rpm")
	if err != nil {
		return fmt.Errorf("Failed to install new package on %s: %s", node, err)
	}

	zap.S().Info("Applying configuration")
	_, _, err = client.RunCommand("mkdir -p /etc/pf9")
	if err != nil {
		return fmt.Errorf("failed to create conf dir on %s: %s", node, err)
	}
	_, _, err = client.RunCommand("tar -zxvf /tmp/pf9config.tgz -C /tmp")
	if err != nil {
		return fmt.Errorf("failed to extract config on %s: %s", node, err)
	}

	_, _, err = client.RunCommand("/bin/cp -R -f /tmp/etc/pf9/* /etc/pf9/")
	if err != nil {
		return fmt.Errorf("failed to copy extracted config to correct location on %s: %s", node, err)
	}

	zap.S().Info("Starting nodelet to bring up the cluster on node: ", node)
	_, _, err = client.RunCommand("/opt/pf9/nodelet/nodeletd phases start")
	if err != nil {
		return fmt.Errorf("failed to start nodelet on %s: %s", node, err)
	}

	zap.S().Info("node successfully upgraded")

	return nil
}
