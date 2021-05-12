// Copyright © 2020 The Platform9 Systems Inc.
package qbert

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	rhttp "github.com/hashicorp/go-retryablehttp"
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
	GetNodePoolID(projectID, token string) (string, error)
	CheckClusterExists(Name, projectID, token string) (bool, string, error)
	Get_converge_status(uuid, projectID, token string) (string, error)
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

type ClusterStatusCheckRequest struct {
	AllowWorkloadsOnMaster    int64  `json:"allowWorkloadsOnMaster"`
	ApiserverStorageBackend   string `json:"apiserverStorageBackend"`
	AppCatalogEnabled         int64  `json:"appCatalogEnabled"`
	AuthzEnabled              int64  `json:"authzEnabled"`
	CalicoIPv4                string `json:"calicoIPv4"`
	CalicoIPv4DetectionMethod string `json:"calicoIPv4DetectionMethod"`
	CalicoIPv6                string `json:"calicoIPv6"`
	CalicoIPv6DetectionMethod string `json:"calicoIPv6DetectionMethod"`
	CalicoIPv6PoolBlockSize   string `json:"calicoIPv6PoolBlockSize"`
	CalicoIPv6PoolCidr        string `json:"calicoIPv6PoolCidr"`
	CalicoIPv6PoolNatOutgoing int64  `json:"calicoIPv6PoolNatOutgoing"`
	CalicoIPIPMode            string `json:"calicoIpIpMode"`
	CalicoNatOutgoing         int64  `json:"calicoNatOutgoing"`
	CalicoRouterID            string `json:"calicoRouterID"`
	CalicoV4BlockSize         string `json:"calicoV4BlockSize"`
	CanMinorUpgrade           int64  `json:"canMinorUpgrade"`
	CanPatchUpgrade           int64  `json:"canPatchUpgrade"`
	CanUpgrade                bool   `json:"canUpgrade"`
	CloudProperties           struct {
		MasterNodes string `json:"masterNodes"`
	} `json:"cloudProperties"`
	CloudProviderName     string `json:"cloudProviderName"`
	CloudProviderType     string `json:"cloudProviderType"`
	CloudProviderUUID     string `json:"cloudProviderUuid"`
	ContainersCidr        string `json:"containersCidr"`
	CPUManagerPolicy      string `json:"cpuManagerPolicy"`
	CreatedAt             string `json:"created_at"`
	Debug                 string `json:"debug"`
	DeployKubevirt        int64  `json:"deployKubevirt"`
	DeployLuigiOperator   int64  `json:"deployLuigiOperator"`
	DockerPrivateRegistry string `json:"dockerPrivateRegistry"`
	DockerRoot            string `json:"dockerRoot"`
	EnableCAS             int64  `json:"enableCAS"`
	EnableMetallb         int64  `json:"enableMetallb"`
	EtcdBackup            struct {
		IntervalInMins      int64 `json:"intervalInMins"`
		IsEtcdBackupEnabled int64 `json:"isEtcdBackupEnabled"`
		StorageProperties   struct {
			LocalPath string `json:"localPath"`
		} `json:"storageProperties"`
		StorageType     string `json:"storageType"`
		TaskErrorDetail string `json:"taskErrorDetail"`
		TaskStatus      string `json:"taskStatus"`
	} `json:"etcdBackup"`
	EtcdDataDir             string      `json:"etcdDataDir"`
	EtcdElectionTimeoutMs   string      `json:"etcdElectionTimeoutMs"`
	EtcdHeartbeatIntervalMs string      `json:"etcdHeartbeatIntervalMs"`
	EtcdVersion             string      `json:"etcdVersion"`
	ExternalDNSName         string      `json:"externalDnsName"`
	FelixIPv6Support        int64       `json:"felixIPv6Support"`
	FlannelIfaceLabel       string      `json:"flannelIfaceLabel"`
	FlannelPublicIfaceLabel string      `json:"flannelPublicIfaceLabel"`
	GcrPrivateRegistry      string      `json:"gcrPrivateRegistry"`
	Ipv6                    int64       `json:"ipv6"`
	IsKubernetes            int64       `json:"isKubernetes"`
	IsMesos                 int64       `json:"isMesos"`
	IsSwarm                 int64       `json:"isSwarm"`
	K8sAPIPort              string      `json:"k8sApiPort"`
	K8sPrivateRegistry      string      `json:"k8sPrivateRegistry"`
	KeystoneEnabled         int64       `json:"keystoneEnabled"`
	KubeProxyMode           string      `json:"kubeProxyMode"`
	KubeRoleVersion         string      `json:"kubeRoleVersion"`
	LastOk                  interface{} `json:"lastOk"`
	LastOp                  string      `json:"lastOp"`
	MasterIP                string      `json:"masterIp"`
	MasterStatus            string      `json:"masterStatus"`
	MasterVipIface          string      `json:"masterVipIface"`
	MasterVipIpv4           string      `json:"masterVipIpv4"`
	MasterVipVrouterID      string      `json:"masterVipVrouterId"`
	Masterless              int64       `json:"masterless"`
	MetallbCidr             string      `json:"metallbCidr"`
	MinorUpgradeRoleVersion string      `json:"minorUpgradeRoleVersion"`
	MtuSize                 string      `json:"mtuSize"`
	Name                    string      `json:"name"`
	NetworkPlugin           string      `json:"networkPlugin"`
	NodePoolName            string      `json:"nodePoolName"`
	NodePoolUUID            string      `json:"nodePoolUuid"`
	NumMasters              int64       `json:"numMasters"`
	NumMaxWorkers           int64       `json:"numMaxWorkers"`
	NumMinWorkers           int64       `json:"numMinWorkers"`
	NumWorkers              int64       `json:"numWorkers"`
	PatchUpgradeRoleVersion string      `json:"patchUpgradeRoleVersion"`
	Privileged              int64       `json:"privileged"`
	ProjectID               string      `json:"projectId"`
	QuayPrivateRegistry     string      `json:"quayPrivateRegistry"`
	ReservedCPUs            string      `json:"reservedCPUs"`
	RuntimeConfig           string      `json:"runtimeConfig"`
	ServicesCidr            string      `json:"servicesCidr"`
	Status                  string      `json:"status"`
	Tags                    struct {
		Pf9_system_monitoring string `json:"pf9-system:monitoring"`
	} `json:"tags"`
	TaskError             interface{} `json:"taskError"`
	TaskStatus            string      `json:"taskStatus"`
	TopologyManagerPolicy string      `json:"topologyManagerPolicy"`
	UpgradingTo           interface{} `json:"upgradingTo"`
	UUID                  string      `json:"uuid"`
	WorkerStatus          string      `json:"workerStatus"`
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

	client := http.Client{}

	req, err := http.NewRequest("POST", attachEndpoint, strings.NewReader(string(byt)))
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
		return fmt.Errorf("Unable to attach node to cluster, code: %d", resp.StatusCode)
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

func (c QbertImpl) Get_converge_status(uuid, projectID, token string) (string, error) {
	qbertApiClustersEndpoint := fmt.Sprintf("%s/qbert/v3/%s/clusters/%s", c.fqdn, projectID, uuid) // Context should return projectID,make changes to keystoneAuth.
	client := rhttp.Client{}
	client.RetryMax = 5
	client.CheckRetry = rhttp.CheckRetry(util.RetryPolicyOn404)
	client.Logger = &util.ZapWrapper{}
	req, err := rhttp.NewRequest("GET", qbertApiClustersEndpoint, nil)

	if err != nil {
		return "", fmt.Errorf("Unable to create request to check cluster status: %w", err)
	}

	req.Header.Set("X-Auth-Token", token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Unable to create request to check cluster status: %w", err)
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Unable to create request to check cluster status: %w", err)
	}
	clusterStatusCheckRequest := ClusterStatusCheckRequest{}
	err = json.NewDecoder(resp.Body).Decode(&clusterStatusCheckRequest)
	if err != nil {
		zap.S().Errorf("Failed to decode clusterStatusCheckRequest, Error: %s", err)
	}

	return clusterStatusCheckRequest.Status, nil
}
