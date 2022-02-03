// Copyright © 2020 The Platform9 Systems Inc.
package qbert

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/platform9/pf9ctl/pkg/util"
	"go.uber.org/zap"
)

// CloudProviderType specifies the infrastructure where the cluster runs
type CloudProviderType string

// CNIBackend specifies the networking solution used for the k8s cluster
type CNIBackend string

type Qbert interface {
	CreateCluster(r ClusterCreateRequest, projectID, token string) (string, error)
	AttachNode(clusterID, projectID, token string, nodeIDs []string, nodetype string) error
	DetachNode(clusterID, projectID, token string, nodeID string) error
	DeleteCluster(clusterID, projectID, token string) error
	DeauthoriseNode(nodeUuid, token string) error
	AuthoriseNode(nodeUuid, token string) error
	GetNodePoolID(projectID, token string) (string, error)
	CheckClusterExists(Name, projectID, token string) (bool, string, error)
	CheckClusterExistsWithUuid(uuid, projectID, token string) (string, error)
}

func NewQbert(fqdn string) Qbert {
	return QbertImpl{fqdn}
}

type QbertImpl struct {
	fqdn string
}

type ClusterCreateRequest struct {
	Name                  string     `json:"name"`
	ContainerCIDR         string     `json:"containersCidr"`
	ServiceCIDR           string     `json:"servicesCidr"`
	MasterVirtualIP       string     `json:"masterVipIpv4"`
	MasterVirtualIPIface  string     `json:"masterVipIface"`
	AllowWorkloadOnMaster bool       `json:"allowWorkloadsOnMaster"`
	Privileged            bool       `json:"privileged"`
	ExternalDNSName       string     `json:"externalDnsName"`
	NetworkPlugin         CNIBackend `json:"networkPlugin"`
	MetalLBAddressPool    string     `json:"metallbCidr"`
	NodePoolUUID          string     `json:"nodePoolUuid"`
	EnableMetalLb         bool       `json:"enableMetallb"`
	Masterless            bool       `json:"masterless"`
}

func (c QbertImpl) CreateCluster(
	r ClusterCreateRequest,
	projectID, token string) (string, error) {

	exists, _, err := c.CheckClusterExists(r.Name, projectID, token)

	if err != nil {
		return "", fmt.Errorf("Unable to check existing cluster: %s", err.Error())
	}

	if exists {
		return "", fmt.Errorf("Cluster name already exists, please select a different name while cluster creation")
	}

	np, err := c.GetNodePoolID(projectID, token)
	if err != nil {
		return "", fmt.Errorf("Unable to fetch nodepoolUuid: %s", err.Error())
	}

	r.NodePoolUUID = np
	byt, err := json.Marshal(r)
	if err != nil {
		return "", fmt.Errorf("Unable to marshal payload: %s", err.Error())
	}

	url := fmt.Sprintf("%s/qbert/v3/%s/clusters", c.fqdn, projectID)

	client := http.Client{}
	req, err := http.NewRequest("POST", url, strings.NewReader(string(byt)))

	if err != nil {
		fmt.Println(err.Error())
		return "", err
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

	var payload map[string]string
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&payload)
	if err != nil {
		return "", err
	}

	return payload["uuid"], nil
}

func (c QbertImpl) AttachNode(clusterID, projectID, token string, nodeIDs []string, nodetype string) error {
	zap.S().Debugf("Attaching the node: %s to cluster: %s", nodeIDs, clusterID)

	var p []map[string]interface{}

	for _, nodeid := range nodeIDs {
		if nodetype == "master" {
			p = append(p, map[string]interface{}{
				"uuid":     nodeid,
				"isMaster": true,
			})
		} else {
			p = append(p, map[string]interface{}{
				"uuid":     nodeid,
				"isMaster": false,
			})
		}
	}

	byt, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("Unable to marshal payload: %s", err.Error())
	}

	attachEndpoint := fmt.Sprintf(
		"%s/qbert/v3/%s/clusters/%s/attach",
		c.fqdn, projectID, clusterID)

	resp, err := Attach_Status(attachEndpoint, token, byt)
	if err != nil {
		return err
	}

	LoopVariable := 1
	for LoopVariable <= util.MaxRetryValue {
		if resp.StatusCode != 200 {
			time.Sleep(30 * time.Second)
			zap.S().Debug("Trying to attach-node to cluster")
			resp, err = Attach_Status(attachEndpoint, token, byt)
			if err != nil {
				return err
			}
			LoopVariable = LoopVariable + 1
		} else {
			break
		}
	}
	if resp.StatusCode != 200 {
		respString, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			zap.S().Errorf("Error occured while converting response body to string")
		}
		zap.S().Debug(string(respString))
		return fmt.Errorf("%v", string(respString))
	}
	return nil
}

func (c QbertImpl) DetachNode(clusterID, projectID, token string, nodeID string) error {
	zap.S().Debugf("Detaching the node: %s from cluster: %s", nodeID, clusterID)

	var p []map[string]interface{}

	p = append(p, map[string]interface{}{
		"uuid": nodeID,
	})

	byt, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("Unable to marshal payload: %s", err.Error())
	}

	detachEndpoint := fmt.Sprintf(
		"%s/qbert/v3/%s/clusters/%s/detach",
		c.fqdn, projectID, clusterID)

	client := http.Client{}

	req, err := http.NewRequest("POST", detachEndpoint, strings.NewReader(string(byt)))
	if err != nil {
		return fmt.Errorf("Unable to create a request: %w", err)
	}
	req.Header.Set("X-Auth-Token", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Unable to POST request through client: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respString, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			zap.S().Info("Error occured while converting response body to string")
		}
		zap.S().Debug(string(respString))
		return fmt.Errorf("%v", string(respString))
	}
	return nil
}

func (c QbertImpl) DeleteCluster(clusterID, projectID, token string) error {
	zap.S().Debugf("Deleting the %s cluster: ", clusterID)

	deleteEndpoint := fmt.Sprintf(
		"%s/qbert/v3/%s/clusters/%s",
		c.fqdn, projectID, clusterID)

	client := http.Client{}

	req, err := http.NewRequest("DELETE", deleteEndpoint, strings.NewReader(""))
	if err != nil {
		return fmt.Errorf("Unable to create a request: %w", err)
	}
	req.Header.Set("X-Auth-Token", token)
	//req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Unable to DELETE request through client: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respString, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			zap.S().Info("Error occured while converting response body to string")
		}
		zap.S().Debug(string(respString))
		return fmt.Errorf("%v", string(respString))
	}
	return nil
}

func (c QbertImpl) DeauthoriseNode(nodeUuid, token string) error {
	zap.S().Debugf("Deauthorising the %s node: ", nodeUuid)

	deleteEndpoint := fmt.Sprintf(
		"%s/resmgr/v1/hosts/%s",
		c.fqdn, nodeUuid)

	client := http.Client{}

	req, err := http.NewRequest("DELETE", deleteEndpoint, strings.NewReader(""))
	if err != nil {
		return fmt.Errorf("Unable to create a request: %w", err)
	}
	req.Header.Set("X-Auth-Token", token)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Unable to DELETE request through client: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respString, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			zap.S().Info("Error occured while converting response body to string")
		}
		zap.S().Debug(string(respString))
		return fmt.Errorf("%v", string(respString))
	}
	return nil
}

func (c QbertImpl) AuthoriseNode(nodeUuid, token string) error {
	zap.S().Debugf("Authorising the %s node: ", nodeUuid)

	deleteEndpoint := fmt.Sprintf(
		"%s/resmgr/v1/hosts/%s/roles/pf9-kube",
		c.fqdn, nodeUuid)

	client := http.Client{}

	req, err := http.NewRequest("PUT", deleteEndpoint, strings.NewReader(""))
	if err != nil {
		return fmt.Errorf("Unable to create a request: %w", err)
	}
	req.Header.Set("X-Auth-Token", token)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Unable to PUT request through client: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respString, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			zap.S().Info("Error occured while converting response body to string")
		}
		zap.S().Debug(string(respString))
		return fmt.Errorf("%v", string(respString))
	}
	return nil
}

func (c QbertImpl) GetNodePoolID(projectID, token string) (string, error) {

	qbertAPIEndpoint := fmt.Sprintf("%s/qbert/v3/%s/cloudProviders", c.fqdn, projectID) // Context should return projectID,make changes to keystoneAuth.
	client := http.Client{}

	req, err := http.NewRequest("GET", qbertAPIEndpoint, nil)

	if err != nil {
		return "", err
	}

	req.Header.Set("X-Auth-Token", token) //
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Couldn't query the qbert Endpoint: %s", err.Error())
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

func (c QbertImpl) CheckClusterExists(name, projectID, token string) (bool, string, error) {
	qbertApiClustersEndpoint := fmt.Sprintf("%s/qbert/v3/%s/clusters", c.fqdn, projectID) // Context should return projectID,make changes to keystoneAuth.
	client := http.Client{}
	req, err := http.NewRequest("GET", qbertApiClustersEndpoint, nil)

	if err != nil {
		return false, "", fmt.Errorf("Unable to create request to check cluster name: %w", err)
	}

	req.Header.Set("X-Auth-Token", token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return false, "", err
	}
	if resp.StatusCode != 200 {
		return false, "", fmt.Errorf("Couldn't query the qbert Endpoint: %d", resp.StatusCode)
	}
	var payload []map[string]interface{}

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&payload)
	if err != nil {
		return false, "", err
	}

	for _, val := range payload {
		if val["name"] == name {
			cluster_uuid := val["uuid"].(string)
			return true, cluster_uuid, nil
		}
	}

	return false, "", nil
}

func (c QbertImpl) CheckClusterExistsWithUuid(uuid, projectID, token string) (string, error) {
	qbertApiClustersEndpoint := fmt.Sprintf("%s/qbert/v3/%s/clusters/%s", c.fqdn, projectID, uuid)

	client := http.Client{}
	req, err := http.NewRequest("GET", qbertApiClustersEndpoint, nil)

	if err != nil {
		return "", fmt.Errorf("Unable to create request to check cluster name: %w", err)
	}

	req.Header.Set("X-Auth-Token", token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode == 400 {
		return "", fmt.Errorf("cluster with given uuid does not exist: %d", resp.StatusCode)
	} else if resp.StatusCode != 200 {
		return "", fmt.Errorf("could not query the qbert endpoint: %d", resp.StatusCode)
	}

	var payload map[string]interface{}

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&payload)
	if err != nil {
		return "", err
	}

	if payload["uuid"] == uuid {
		cluster_name := payload["name"].(string)
		return cluster_name, nil
	}

	return fmt.Sprintf("error finding cluster with uuid %s", uuid), nil

}

//Function to Check status of attach-node API
func Attach_Status(attachEndpoint string, token string, byt []byte) (*http.Response, error) {
	client := http.Client{}
	req, err := http.NewRequest("POST", attachEndpoint, strings.NewReader(string(byt)))
	if err != nil {
		zap.S().Debugf("Unable to create a request: ", err)
		return nil, fmt.Errorf("Unable to create a request: %w", err)
	}
	req.Header.Set("X-Auth-Token", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		zap.S().Debugf("Unable to POST request through client: ", err)
		return resp, fmt.Errorf("Unable to POST request through client: %w", err)
	}

	defer resp.Body.Close()
	return resp, nil
}
