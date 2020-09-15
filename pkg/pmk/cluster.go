package pmk

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// Cluster defines Kubernetes cluster
type Cluster struct {
	Name                  string
	ContainerCIDR         string
	ServiceCIDR           string
	MasterVirtualIP       string
	MasterVirtualIPIface  string
	AllowWorkloadOnMaster bool
	Privileged            bool
	CloudProvider         CloudProviderType
	ExternalDNSName       string
	NetworkPlugin         CNIBackend
	MetalLBAddressPool    string
	CloudProviderParams   interface{}
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
	}, nil
}

// Create a cluster in the management plan.
func (c Cluster) Create(ctx Context, auth KeystoneAuth) error {
	return nil
}

// Exists checks if the cluster with the same name
// exists or not.
func (c Cluster) Exists(name string) (bool, string) {
	return false, ""
}

// type CloudProvider struct {
// 	name         string
// 	Providertype string
// 	uuid         string
// 	nodePoolUuid string
// }

// TODO: Piyush ?
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
		fmt.Errorf("Couldn't query the qbert Endpoint: %s", err.Error())
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
