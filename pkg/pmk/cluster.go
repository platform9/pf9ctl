package pmk

import (
	"fmt"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/platform9/pf9ctl/pkg/client"
	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/keystone"
	"github.com/platform9/pf9ctl/pkg/objects"
	"github.com/platform9/pf9ctl/pkg/platform"
	"github.com/platform9/pf9ctl/pkg/platform/centos"
	"github.com/platform9/pf9ctl/pkg/platform/debian"
	"github.com/platform9/pf9ctl/pkg/qbert"
	"github.com/platform9/pf9ctl/pkg/util"
	"go.uber.org/zap"
)

// Bootstrap simply onboards the local node and attaches it as master to a newly created cluster.
func Bootstrap(ctx objects.Config, c client.Client, req qbert.ClusterCreateRequest, keystoneAuth keystone.KeystoneAuth, bootConfig objects.NodeConfig) error {

	if err1 := c.Segment.SendEvent("Starting Cluster creation(Bootstrap)", keystoneAuth, checkPass, ""); err1 != nil {
		zap.S().Debugf("Unable to send Segment event for bootstrap node. Error: %s", err1.Error())
	}

	token := keystoneAuth.Token
	clustername := fmt.Sprintf(" Creating a cluster %s", req.Name)
	zap.S().Debug(clustername)
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Color("red")
	s.Start() // Start the spinner
	defer s.Stop()
	s.Suffix = clustername

	clusterID, err := c.Qbert.CreateCluster(
		req,
		keystoneAuth.ProjectID,
		keystoneAuth.Token)

	s.Stop()

	if err != nil {
		fmt.Println(color.Red("x")+" Unable to create cluster. Error:", err)
		zap.S().Debug("Unable to create cluster. Error:", err)
		if err = c.Segment.SendEvent("Cluster creation(Bootstrap)", keystoneAuth, checkFail, ""); err != nil {
			zap.S().Debugf("Unable to send Segment event for bootstrap node. Error: %s", err.Error())
		}
		return fmt.Errorf("Unable to create cluster " + req.Name)
	}

	fmt.Println(color.Green("✓") + " Cluster creation completed")
	zap.S().Debug("Cluster creation completed")
	if err = c.Segment.SendEvent("Cluster creation(Bootstrap)", keystoneAuth, checkPass, ""); err != nil {
		zap.S().Debugf("Unable to send Segment event for bootstrap node. Error: %s", err.Error())
	}

	s.Color("red")
	s.Start() // Start the spinner
	defer s.Stop()
	s.Suffix = " Checking Host Status"
	zap.S().Debug("Checking Host Status")
	cmd := `grep ^host_id /etc/pf9/host_id.conf | cut -d = -f2 | cut -d ' ' -f2`
	output, err := c.Executor.RunWithStdout("bash", "-c", cmd)
	if err != nil {
		return fmt.Errorf("Unable to execute command: %w", err)
	}
	nodeID := strings.TrimSpace(string(output))

	LoopVariable := 1
	for LoopVariable <= util.MaxLoopValue {
		hostStatus := c.Resmgr.HostStatus(token, nodeID)
		if !hostStatus {
			zap.S().Debugf("Host is Down...Trying again")
		} else {
			util.HostDown = false
			break
		}
		LoopVariable = LoopVariable + 1
	}

	if LoopVariable > util.MaxLoopValue {
		util.HostDown = true
	}

	s.Stop()

	if !util.HostDown {

		zap.S().Debugf("Host is connected")
		fmt.Println(color.Green("✓") + " Host is connected")
		if err = c.Segment.SendEvent("Host Connected(Bootstrap)", keystoneAuth, checkPass, ""); err != nil {
			zap.S().Debugf("Unable to send Segment event for bootstrap node. Error: %s", err.Error())
		}
	} else {
		fmt.Println(color.Red("x") + " Host is disconnected. Unable to attach this node to the cluster " + req.Name + " Run prep-node/authorize-node and try again")
		zap.S().Debug("Host is disconnected. Unable to attach this node to the cluster " + req.Name + " Run prep-node/authorize-node and try again")
		if err = c.Segment.SendEvent("Host Connected(Bootstrap)", keystoneAuth, checkFail, ""); err != nil {
			zap.S().Debugf("Unable to send Segment event for bootstrap node. Error: %s", err.Error())
		}
		//Deleting the cluster if the host is disconnected
		DeleteClusterBootstrap(clusterID, c, keystoneAuth, token)
		return fmt.Errorf("Host is disconnected. Unable to attach this node to the cluster " + req.Name + " Run prep-node/authorize-node and try again")
	}

	attachname := fmt.Sprintf(" Attaching node to the cluster %s", req.Name)
	s.Color("red")
	s.Start() // Start the spinner
	defer s.Stop()
	s.Suffix = attachname
	zap.S().Debug(attachname)
	time.Sleep(30 * time.Second)
	var nodeIDs []string
	nodeIDs = append(nodeIDs, nodeID)

	err = c.Qbert.AttachNode(
		clusterID,
		keystoneAuth.ProjectID, keystoneAuth.Token, nodeIDs, "master")

	s.Stop() //Stop the Spinner

	if err != nil {
		fmt.Println(color.Red("x") + " Unable to attach-node to cluster " + req.Name + "Run bootstrap again")
		zap.S().Debug("Unable to attach-node to cluster. Error:", err)
		if err = c.Segment.SendEvent("Attach-Node(Bootstrap)", keystoneAuth, checkFail, ""); err != nil {
			zap.S().Debugf("Unable to send Segment event for bootstrap node. Error: %s", err.Error())
		}

		//Deleting the cluster if the node is not attached to the cluster
		DeleteClusterBootstrap(clusterID, c, keystoneAuth, token)
		zap.S().Debug("Unable to attach node to cluster " + req.Name + "Run bootstrap again")
		return fmt.Errorf("Unable to attach node to cluster " + req.Name + "Run bootstrap again")
	}

	fmt.Println(color.Green("✓") + " Attached node to the cluster")
	zap.S().Debug("Attached node to the cluster")
	if err = c.Segment.SendEvent("Attach-Node(Bootstrap)", keystoneAuth, checkPass, ""); err != nil {
		zap.S().Debugf("Unable to send Segment event for bootstrap node. Error: %s", err.Error())
	}

	if err = c.Segment.SendEvent("Bootstrap Completed Successfully", keystoneAuth, checkPass, ""); err != nil {
		zap.S().Debugf("Unable to send Segment event for bootstrap node. Error: %s", err.Error())
	}
	fmt.Println(color.Green("✓") + " Bootstrap successfully finished")
	zap.S().Debug("Bootstrap successfully finished")
	zap.S().Debug("Cluster creation started....This may take a few minutes....Check the latest status in UI")
	fmt.Println("Cluster creation started....This may take a few minutes....Check the latest status in UI")
	return nil
}

//Checks Prerequisites for Bootstrap Command
func PreReqBootstrap(executor cmdexec.Executor) (bool, bool, error) {

	os, err := ValidatePlatform(executor)
	if err != nil {
		zap.S().Fatalf("OS version is not supported")
	}

	var Instance platform.Platform
	if os == "debian" {
		Instance = debian.NewDebian(executor)

	} else if os == "redhat" {
		Instance = centos.NewCentOS(executor)
	} else {
		zap.S().Infof("OS version is not supported")
		return false, false, fmt.Errorf("OS version is not supported")
	}

	val, err := Instance.CheckExistingInstallation()
	if err != nil {
		//zap.S().Fatalf("Could not run command Installation")
		zap.S().Fatalf("Error %s", err)
	}

	val1, err1 := Instance.CheckKubernetesCluster()
	if err1 != nil {
		//zap.S().Fatalf("Could not run command Cluster")
		zap.S().Fatalf("Error %s", err1)
	}
	return val, val1, nil
}

//Deleting the cluster if the node is not attached to the cluster
func DeleteClusterBootstrap(clusterID string, c client.Client, keystoneAuth keystone.KeystoneAuth, token string) {
	err := c.Qbert.DeleteCluster(clusterID, keystoneAuth.ProjectID, token)

	if err != nil {
		if err = c.Segment.SendEvent("Delete Cluster(Bootstrap)", keystoneAuth, checkFail, ""); err != nil {
			zap.S().Debugf("Unable to send Segment event for bootstrap node. Error: %s", err.Error())
		}
		zap.S().Debugf("Unable to delete cluster")
	} else {
		if err = c.Segment.SendEvent("Delete Cluster(Bootstrap)", keystoneAuth, checkPass, ""); err != nil {
			zap.S().Debugf("Unable to send Segment event for bootstrap node. Error: %s", err.Error())
		}
		zap.S().Debugf("Deleted the cluster successfully")
	}
}
