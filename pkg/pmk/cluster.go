package pmk

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// Cluster defines Kubernetes cluster
type Cluster struct {
	Name                  string            `json:"name"`
	UUID                  string            `json:"-"`
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
) (*Cluster, error) {

	return &Cluster{
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
func (c *Cluster) Create(ctx Context, auth KeystoneAuth) (string, error) {
	log.Println("Received a call to create a cluster in management plane")

	np, err := c.getNodePoolUUID(ctx, auth)
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

	// Setin the UUID for the cluster
	// achieved by creating the cluster.
	c.UUID = payload["uuid"]

	return payload["uuid"], nil
}

// Exists checks if the cluster with the same name
// exists or not.
func (c Cluster) Exists(name string) (bool, string) {
	return false, ""
}

// AttachNode attaches a node onto the cluster/
func (c *Cluster) AttachNode(ctx Context, auth KeystoneAuth, nodeUUID string) error {
	log.Printf("Received a call to attachnode: %s to cluster: %s\n",
		nodeUUID, c.UUID)

	var p []map[string]interface{}
	p = append(p, map[string]interface{}{
		"uuid":     nodeUUID,
		"isMaster": true,
	})

	byt, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("Unable to marshal payload: %s", err.Error())
	}

	attachEndpoint := fmt.Sprintf(
		"%s/qbert/v3/%s/clusters/%s/attach",
		ctx.Fqdn, auth.ProjectID, c.UUID)

	client := http.Client{}

	req, err := http.NewRequest("POST", attachEndpoint, strings.NewReader(string(byt)))
	req.Header.Set("X-Auth-Token", auth.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("Unable to attach node, respCode: %d", resp.StatusCode)
	}

	return nil
}

// GetNodePoolUUID fetches the nodepooluuid
func (c Cluster) getNodePoolUUID(ctx Context, keystoneAuth KeystoneAuth) (string, error) {

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

}
