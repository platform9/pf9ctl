package pmk

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

// Cluster defines Kubernetes cluster
type Cluster struct {
	Name                  string            `json:"name"`
	ContainerCIDR         string            `json:"containersCidr"`
	ServiceCIDR           string            `json:"servicesCidr"`
	MasterVirtualIP       string            `json:"masterVipIpv4"`
	MasterVirtualIPIface  string            `json:"masterVipIface"`
	AllowWorkloadOnMaster bool              `json:"allowWorkloadsOnMaster"`
	Privileged            bool              `json:"privileged"`
	CloudProvider         CloudProviderType `json:"-"`
	ExternalDNSName       string            `json:"externalDnsName"`
	NetworkPlugin         CNIBackend        `json:"networkPlugin"`
	MetalLBAddressPool    string            `json:"metallbCidr"`
	NodePoolUUID          string            `json:"nodePoolUuid"`
	EnableMetalLb         bool              `json:"enableMetallb"`
	Masterless            bool              `json:"masterless"`
}

// NewClusterCreate returns an instance of new cluster.
func NewClusterCreate(
	name,
	containerCidr,
	serviceCidr,
	masterVIP,
	masterVIPIface,
	externalDNSName,
	networkPlugin,
	metallbAddressPool string,
	allowWorkloadsOnMaster,
	privileged bool,
) (Cluster, error) {

	return Cluster{
		Name:                  name,
		ContainerCIDR:         containerCidr,
		ServiceCIDR:           serviceCidr,
		MasterVirtualIP:       masterVIP,
		MasterVirtualIPIface:  masterVIPIface,
		AllowWorkloadOnMaster: allowWorkloadsOnMaster,
		Privileged:            privileged,
		NetworkPlugin:         CNIBackend(networkPlugin),
		ExternalDNSName:       externalDNSName,
		MetalLBAddressPool:    metallbAddressPool,
		EnableMetalLb:         metallbAddressPool != "",
	}, nil
}

// Create a cluster in the management plan.
func (c Cluster) Create(ctx Context, auth KeystoneAuth) (string, error) {
	log.Println("Received a call to create a cluster in management plane")

	np, err := GetNodePoolUUID(ctx, auth)
	if err != nil {
		return "", fmt.Errorf("Unable to fetch nodepoolUuid: %s", err.Error())
	}

	c.NodePoolUUID = np
	byt, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("Unable to marshal payload: %s", err.Error())
	}

	url := fmt.Sprintf("%s/qbert/v3/%s/clusters", ctx.Fqdn, auth.ProjectID)
	client := http.Client{}

	req, err := http.NewRequest("POST", url, strings.NewReader(string(byt)))

	if err != nil {
		fmt.Println(err.Error())
		return "", err
	}

	req.Header.Set("X-Auth-Token", auth.Token)
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

	for _, val := range payload {
		return val, err

	}
	return "", err
}

// Exists checks if the cluster with the same name
// exists or not.
func (c Cluster) Exists(name string) (bool, string) {
	return false, ""
}

// GetNodePoolUUID fetches the nodepooluuid
func GetNodePoolUUID(ctx Context, keystoneAuth KeystoneAuth) (string, error) {

	qbertAPIEndpoint := fmt.Sprintf("%s/qbert/v3/%s/cloudProviders", ctx.Fqdn, keystoneAuth.ProjectID) // Context should return projectID,make changes to keystoneAuth.
	fmt.Println(qbertAPIEndpoint)

	client := http.Client{}

	req, err := http.NewRequest("GET", qbertAPIEndpoint, nil)

	if err != nil {
		fmt.Println(err.Error())
		return "", err
	}

	req.Header.Set("X-Auth-Token", keystoneAuth.Token) //
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

	//fmt.Println(cloudProviderData)

}

func AttachNodeBootStrap(clus_uuid string, ctx Context, auth KeystoneAuth) error {

	cmd := `cat /etc/pf9/host_id.conf | grep ^host_id | cut -d = -f2 | cut -d ' ' -f2`
	nodeUUID, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		fmt.Errorf("Host_id.conf not found: %s", err.Error())
	}

	nodeStruct := fmt.Sprintf(`{
	"uuid": "%s"
	"isMaster" : true	
	}`, nodeUUID)
	attachEndpoint := fmt.Sprintf("%s/qbert/v3/%s/clusters/%s/attach", ctx.Fqdn, auth.ProjectID)
	fmt.Println(attachEndpoint)

	resp, err := http.Post(attachEndpoint, "application/json", strings.NewReader(nodeStruct))
	if err != nil {
		return err
	}

	if resp.StatusCode != 201 {
		return fmt.Errorf("Unable to get keystone token, status: %d", resp.StatusCode)
	}

	return err
}
