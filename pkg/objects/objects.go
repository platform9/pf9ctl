package objects

import "time"

// objects stores information to contact with the pf9 controller.

type NodeConfig struct {
	APIVersion string `yaml:"apiVersion" json:"apiVersion"`
	Kind       string `yaml:"kind" json:"kind"`
	Spec       Spec   `yaml:"spec" json:"spec"`
	SshKey     string `yaml:"ssh-key" json:"ssh-key"`
	Password   string `yaml:"password,omitempty" json:"password,omitempty"`
}
type Node struct {
	Ip       string `yaml:"ip" json:"ip"`
	Hostname string `yaml:"hostname" json:"hostname"`
	Type     string `yaml:"type" json:"type"`
}
type Spec struct {
	DeploymentKind string `yaml:"deploymentKind" json:"deploymentKind"`
	Type           string `yaml:"type" json:"type"`
	Nodes          []Node `yaml:"nodes" json:"nodes"`
}

type UserAWSCreds struct {
	AwsIamUsername string `yaml:"aws_iam_username" json:"aws_iam_username"`
	AwsAccessKey   string `yaml:"aws_access_key" json:"aws_access_key"`
	AwsSecretKey   string `yaml:"aws_secret_key" json:"aws_secret_key"`
	AwsRegion      string `yaml:"aws_region" json:"aws_region"`
}

type UserAzureCreds struct {
	AzureTenant       string `yaml:"azure_tenant" json:"azure_tenant"`
	AzureClient       string `yaml:"azure_application" json:"azure_application"`
	AzureSubscription string `yaml:"azure_subscription" json:"azure_subscription"`
	AzureSecret       string `yaml:"azure_secret" json:"azure_secret"`
}

type UserGoogleCreds struct {
	GooglePath         string `yaml:"google_path" json:"google_path"`
	GoogleProjectName  string `yaml:"google_project_name" json:"google_project_name"`
	GoogleServiceEmail string `yaml:"google_service_email" json:"google_service_email"`
}

type Other struct {
	WaitPeriod    time.Duration `yaml:"wait_period" json:"wait_period"`
	AllowInsecure bool          `yaml:"allow_insecure" json:"allow_insecure"`
}

type UserData struct {
	AccountUrl string          `yaml:"account_url" json:"account_url"`
	Username   string          `yaml:"username" json:"username"`
	Password   string          `yaml:"password" json:"password"`
	Tenant     string          `yaml:"tenant" json:"tenant"`
	Region     string          `yaml:"region" json:"redion"`
	MfaToken   string          `yaml:"mfa_token" json:"maf_token"`
	ProxyURL   string          `yaml:"proxy_url" json:"proxy_url"`
	AWS        UserAWSCreds    `yaml:"user_aws_creds" json:"user_aws_creds"`
	Azure      UserAzureCreds  `yaml:"user_azure_creds" json:"user_azure_creds"`
	Google     UserGoogleCreds `yaml:"user_google_creds" json:"user_google_creds"`
	OtherData  Other           `yaml:"other" json:"other"`
}

type Config struct {
	ApiVersion string   `yaml:"apiVersion" json:"apiVersion"`
	Kind       string   `yaml:"kind" json:"kind"`
	Spec       UserData `yaml:"spec" json:"spec"`
}

type ClusterConfig struct {
	APIVersion string      `yaml:"APIVersion"`
	Kind       string      `yaml:"Kind"`
	Spec       ClusterSpec `yaml:"Spec"`
}

type ClusterSpec struct {
	DeploymentKind              string                `yaml:"Deployment Kind"`
	ClusterSetting              ClusterInfo           `yaml:"Cluster Setting"`
	ApplicationContainerSetting ContainerSetting      `yaml:"Application & Container Settings"`
	NetworkingAndRegistration   Networking            `yaml:"Networking & Registration"`
	MasterNodes                 []MasterNode          `yaml:"Master Nodes"`
	WorkerNodes                 []WorkerNode          `yaml:"Worker Nodes"`
	ClusterAddOns               ClusterAddon          `yaml:"Cluster Add-Ons"`
	ClusterNetworkInfo          ClusterNetworkInfo    `yaml:"Cluster Network Info"`
	AdvancedConfiguration       AdvancedConfiguration `yaml:"Advanced Configuration"`
}

type MasterNode struct {
	IP       string `yaml:"IP"`
	HostName string `yaml:"Host Name"`
}

type WorkerNode struct {
	IP       string `yaml:"IP"`
	HostName string `yaml:"Host Name"`
}

type ClusterInfo struct {
	Name            string `yaml:"Name"`
	KubeRoleVersion string `yaml:"Kube Role Version"`
}

type ContainerSetting struct {
	Previleged            bool   `yaml:"Previleged"`
	AllowWorkloadOnMaster bool   `yaml:"Make Master nodes Master + Worker"`
	ContainerRuntime      string `yaml:"Container Runtime"`
}

type Networking struct {
	ClusterNetworkStack NetworkStack `yaml:"Cluster Network Stack"`
	NodeRegistration    Registration `yaml:"Node Registration"`
}

type NetworkStack struct {
	IPv4 bool `yaml:"IPv4 Networking Stack"`
	IPv6 bool `yaml:"IPv6 Networking Stack"`
}

type Registration struct {
	UseNodeIPForClusterCreation       bool `yaml:"Use Node IP address for Cluster Creation"`
	UseNodeHostNameForClusterCreation bool `yaml:"Use Node Hostname for Cluster Creation"`
}

type ClusterAddon struct {
	EtcdBackup          EtcdBackup `yaml:"ETCD Backup Configuration"`
	DeployKubevirt      bool       `yaml:"Enable KubeVirt"`
	DeployLuigiOperator bool       `yaml:"Enable Network Plugin Operator"`
	MetallbCidr         string     `yaml:"Metallb Address Pool Range"`
	Metal3              Metal3     `yaml:"Metal3"`
	Monitoring          Monitoring `yaml:"Prometheus Monitoring"`
	EnableProfileAgent  bool       `yaml:"Enable Profile Agent"`
}

type EtcdBackup struct {
	StorageProperties       StorageProperties `yaml:"Storage Properties"`
	DailyBackupTime         string            `yaml:"Daily Backup Time"`
	MaxTimestampBackupCount int               `yaml:"Max Timestamp Backup Count"`
	IntervalInHours         int               `yaml:"Interval In Hours"`
	MaxIntervalBackupCount  int               `yaml:"Max Interval BackupCount"`
}

type StorageProperties struct {
	LocalPath string `yaml:"Storage Path"`
}

type Monitoring struct {
	RetentionTime string `yaml:"Retention Time (days)"`
}

type Metal3 struct {
}

type ClusterNetworkInfo struct {
	APIFqdn       string      `yaml:"API FQDN"`
	ContainerCIDR string      `yaml:"Container CIDR"`
	ServiceCIDR   string      `yaml:"Service CIDR"`
	HTTPProxy     string      `yaml:"HTTP Proxy"`
	ClusterCNI    ClusterCNI  `yaml:"Cluster CNI"`
	NatOutgoing   NatOutgoing `yaml:"NAT Outgoing"`
}

type ClusterCNI struct {
	NetworkBackend     string `yaml:"Network Backend"`
	IPEncapsulation    string `yaml:"IP in IP Encapsulation Mode"`
	InterfaceDetection string `yaml:"Interface Detection Method"`
}

type NatOutgoing struct {
	BlockSize string `yaml:"Block Size"`
	MtuSize   string `yaml:"MTU Size"`
}

type AdvancedConfiguration struct {
	APIConfiguration       string   `yaml:"Advanced API Configuration"`
	APIServerFlags         []string `yaml:"API Server Flags"`
	SchedulerFlags         []string `yaml:"Scheduler Flags"`
	ControllerManagerFlags []string `yaml:"Controller Manager Flags"`
	Tags                   Tags     `yaml:"Tags"`
	TopologyManagerPolicy  string   `yaml:"Topology Manager Policy"`
	ReservedCPUs           string   `yaml:"Reserved CPUs"`
}

type Tags struct {
}
