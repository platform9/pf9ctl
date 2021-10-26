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
	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/platform/centos"
	"github.com/platform9/pf9ctl/pkg/platform/debian"
	"github.com/platform9/pf9ctl/pkg/qbert"
	"go.uber.org/zap"
)

const MaxLoopHostStatus = 3

var ex cmdexec.Executor

// Bootstrap simply preps the local node and attaches it as master to a newly created cluster.
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
		return fmt.Errorf("Unable to create cluster"+req.Name+": %w", err)
	}

	//fmt.Printf("ProjectID: %s", keystoneAuth.ProjectID)

	cmd := `cat /etc/pf9/host_id.conf | grep ^host_id | cut -d = -f2 | cut -d ' ' -f2`
	output, err := c.Executor.RunWithStdout("bash", "-c", cmd)
	if err != nil {
		return fmt.Errorf("Unable to get host-id %w", err)
	}
	nodeID := strings.TrimSuffix(string(output), "\n")

	/*LoopVariable := 1
	for LoopVariable <= MaxLoopHostStatus {
		hostStatus := Host_Status(c.Executor, ctx.Fqdn, token, nodeID)
		if hostStatus == "false" {
			zap.S().Debugf("Host is Down...Trying again")
		} else {
			util.HostDown = false
			break
		}
		time.Sleep(20 * time.Second)
		LoopVariable = LoopVariable + 1
	}

	if !util.HostDown {
		zap.S().Debugf("Host is Connected...Proceeding to connect node to cluster " + req.Name)
	} else {
		zap.S().Fatalf("Host is disconnected....Unable to attach this node to the cluster " + req.Name)
	}*/

	time.Sleep(60 * time.Second)
	var nodeIDs []string
	nodeIDs = append(nodeIDs, nodeID)

	s.Stop() //Stop the spinner
	fmt.Println(color.Green("✓") + " Cluster created successfully")
	fmt.Println("Attaching node to the cluster ", req.Name)

	err = c.Qbert.AttachNode(
		clusterID,
		keystoneAuth.ProjectID, keystoneAuth.Token, nodeIDs, "master")

	if err != nil {
		_, err1 := Delete_Cluster(ex, ctx.Fqdn, token, keystoneAuth.ProjectID, clusterID)
		if err1 != nil {
			zap.S().Debug("Deleted the cluster successfully")
		} else {
			zap.S().Debug("Unable to delete the cluster")
		}
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

func CheckClusterNameExists(exec cmdexec.Executor, fqdn string, token string, projectID string, name string) {

}

func PreReqBootstrap(executor cmdexec.Executor) (bool, bool, error) {

	os, err := ValidatePlatform(executor)
	fmt.Println(os)
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
