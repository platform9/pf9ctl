// Copyright © 2020 The Platform9 Systems Inc.

package pmk

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/platform9/pf9ctl/pkg/client"
	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/objects"
	"github.com/platform9/pf9ctl/pkg/platform/centos"
	"github.com/platform9/pf9ctl/pkg/platform/debian"
	"github.com/platform9/pf9ctl/pkg/qbert"
	"github.com/platform9/pf9ctl/pkg/util"
	"go.uber.org/zap"
)

const MaxLoopHostStatus = 3

var ex cmdexec.Executor

// Bootstrap simply preps the local node and attach it as master to a newly
// created cluster.
func Bootstrap(ctx objects.Config, c client.Client, req qbert.ClusterCreateRequest, bootConfig objects.NodeConfig) error {

	keystoneAuth, err := c.Keystone.GetAuth(
		ctx.Username,
		ctx.Password,
		ctx.Tenant,
		ctx.MfaToken,
	)
	if err != nil {
		zap.S().Fatalf("keystone authentication failed: %s", err.Error())
	}

	if err = c.Segment.SendEvent("Starting Cluster creation(Bootstrap)", keystoneAuth, checkPass, ""); err != nil {
		zap.S().Errorf("Unable to send Segment event for bootstrap node. Error: %s", err.Error())
	}

	token := keystoneAuth.Token
	clustername := fmt.Sprintf(" Creating a cluster %s", req.Name)

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Color("red")
	s.Start() // Start the spinner
	defer s.Stop()
	s.Suffix = clustername

	clusterID, err := c.Qbert.CreateCluster(
		req,
		keystoneAuth.ProjectID,
		keystoneAuth.Token)

	if err != nil {
		if err = c.Segment.SendEvent("Cluster creation(Bootstrap)", keystoneAuth, checkFail, ""); err != nil {
			zap.S().Errorf("Unable to send Segment event for bootstrap node. Error: %s", err.Error())
		}
		return fmt.Errorf("Unable to create cluster"+req.Name+": %w", err)
	}

	if err = c.Segment.SendEvent("Cluster creation(Bootstrap)", keystoneAuth, checkPass, ""); err != nil {
		zap.S().Errorf("Unable to send Segment event for bootstrap node. Error: %s", err.Error())
	}

	cmd := `grep ^host_id /etc/pf9/host_id.conf | cut -d = -f2 | cut -d ' ' -f2`
	output, err := c.Executor.RunWithStdout("bash", "-c", cmd)
	if err != nil {
		return fmt.Errorf("Unable to execute command: %w", err)
	}
	nodeID := strings.TrimSuffix(string(output), "\n")

	LoopVariable := 1
	for LoopVariable <= MaxLoopHostStatus {
		hostStatus := Host_Status(c.Executor, ctx.Fqdn, token, nodeID, bootConfig)
		if hostStatus != "true" {
			zap.S().Debugf("Host is Down...Trying again")
		} else {
			util.HostDown = false
			break
		}
		LoopVariable = LoopVariable + 1
	}

	if !util.HostDown {
		zap.S().Debugf("Host is Connected...Proceeding to connect node to cluster " + req.Name)
		if err = c.Segment.SendEvent("Host Connected(Bootstrap)", keystoneAuth, checkPass, ""); err != nil {
			zap.S().Errorf("Unable to send Segment event for bootstrap node. Error: %s", err.Error())
		}
	} else {
		if err = c.Segment.SendEvent("Host Connected(Bootstrap)", keystoneAuth, checkFail, ""); err != nil {
			zap.S().Errorf("Unable to send Segment event for bootstrap node. Error: %s", err.Error())
		}
		_, err1 := Delete_Cluster(ex, ctx.Fqdn, token, keystoneAuth.ProjectID, clusterID)
		if err1 != nil {
			if err = c.Segment.SendEvent("Delete Cluster(Bootstrap)", keystoneAuth, checkFail, ""); err != nil {
				zap.S().Errorf("Unable to send Segment event for bootstrap node. Error: %s", err.Error())
			}
			zap.S().Debugf("Deleted the cluster successfully")
		} else {
			if err = c.Segment.SendEvent("Delete Cluster(Bootstrap)", keystoneAuth, checkPass, ""); err != nil {
				zap.S().Errorf("Unable to send Segment event for bootstrap node. Error: %s", err.Error())
			}
			zap.S().Debugf("Unable to delete the cluster")
		}
		zap.S().Fatalf("Host is disconnected....Unable to attach this node to the cluster " + req.Name)
	}

	time.Sleep(30 * time.Second)
	var nodeIDs []string
	nodeIDs = append(nodeIDs, nodeID)

	s.Stop() //Stop the spinner
	fmt.Println(color.Green("✓") + " Cluster created successfully")
	fmt.Println("Attaching node to the cluster ", req.Name)

	err = c.Qbert.AttachNode(
		clusterID,
		keystoneAuth.ProjectID, keystoneAuth.Token, nodeIDs, "master")

	if err != nil {
		if err = c.Segment.SendEvent("Attach-Node(Bootstrap)", keystoneAuth, checkFail, ""); err != nil {
			zap.S().Errorf("Unable to send Segment event for bootstrap node. Error: %s", err.Error())
		}

		_, err1 := Delete_Cluster(ex, ctx.Fqdn, token, keystoneAuth.ProjectID, clusterID)
		if err1 != nil {
			if err = c.Segment.SendEvent("Delete Cluster(Bootstrap)", keystoneAuth, checkFail, ""); err != nil {
				zap.S().Errorf("Unable to send Segment event for bootstrap node. Error: %s", err.Error())
			}
			zap.S().Debugf("Deleted the cluster successfully")
		} else {
			if err = c.Segment.SendEvent("Delete Cluster(Bootstrap)", keystoneAuth, checkPass, ""); err != nil {
				zap.S().Errorf("Unable to send Segment event for bootstrap node. Error: %s", err.Error())
			}
			zap.S().Debugf("Unable to delete the cluster")
		}
		return fmt.Errorf("Unable to attach node to the cluster"+req.Name+": %w", err)
	}

	if err = c.Segment.SendEvent("Attach-Node(Bootstrap)", keystoneAuth, checkPass, ""); err != nil {
		zap.S().Errorf("Unable to send Segment event for bootstrap node. Error: %s", err.Error())
	}

	if err = c.Segment.SendEvent("Bootstrap Completed Successfully", keystoneAuth, checkPass, ""); err != nil {
		zap.S().Errorf("Unable to send Segment event for bootstrap node. Error: %s", err.Error())
	}
	zap.S().Info("=======Bootstrap successfully finished========")
	return nil
}

//To check the host status before attaching the node to a cluster
func Host_Status(exec cmdexec.Executor, fqdn string, token string, hostID string, bootConfig objects.NodeConfig) string {
	zap.S().Debug("Getting host status")
	isRemote := cmdexec.CheckRemote(bootConfig)

	tkn := fmt.Sprintf(`"X-Auth-Token: %v"`, token)
	cmd := fmt.Sprintf(`curl -sH %v -X GET %v/resmgr/v1/hosts/%v | jq .info.responding `, tkn, fqdn, hostID)
	var status string
	var err1 error

	if isRemote {
		cmnd := fmt.Sprintf(`bash %s`, cmd)
		status, err1 = exec.RunWithStdout(cmnd)
	} else {
		cmnd := fmt.Sprintf(`%s`, cmd)
		status, err1 = exec.RunWithStdout("bash", "-c", cmnd)
	}
	if err1 != nil {
		zap.S().Fatalf("Unable to get host status : ", err1)
	}
	status = strings.TrimSpace(strings.Trim(status, "\n\""))
	zap.S().Debug("Host status is : ", status)
	return status
}

//Delete the Cluster if attachnode fails
func Delete_Cluster(exec cmdexec.Executor, fqdn string, token string, projectID string, clusterID string) (string, error) {
	zap.S().Debug("=====Deleting cluster======")

	qbertApiClustersEndpoint := fmt.Sprintf("%v/qbert/v3/%v/clusters/%v", fqdn, projectID, clusterID)

	client := http.Client{}
	req, err := http.NewRequest("DELETE", qbertApiClustersEndpoint, nil)

	if err != nil {
		return "", fmt.Errorf("Unable to create request to delete cluster : %w", err)
	}

	req.Header.Set("X-Auth-Token", token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Couldn't query the qbert Endpoint: %d", resp.StatusCode)
	}

	var payload []map[string]string

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&payload)
	if err != nil {
		return "", err
	}

	for _, val := range payload {
		if val["type"] == "local" {
			return val["nodePoolUuid"], nil
		}
	}

	return "", errors.New("Unable to locate local Node Pool")
}

//Checks Prerequisites for Bootstrap Command
func PreReqBootstrap(executor cmdexec.Executor) (bool, bool, error) {

	os, err := ValidatePlatform(executor)
	if err != nil {
		zap.S().Fatalf("OS version is not supported")
	}

	if os == "debian" {
		Instance := debian.NewDebian(executor)
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
	} else if os == "redhat" {
		Instance := centos.NewCentOS(executor)
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
	} else {
		zap.S().Infof("OS version is not supported")
		return false, false, fmt.Errorf("OS version is not supported")
	}
}
