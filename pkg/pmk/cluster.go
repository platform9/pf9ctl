package pmk

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

// TODO: Piyush ?
func getNodePoolUUID(ctx Context, keystoneAuth KeystoneAuth) (string, error) {
	return "", nil
}
