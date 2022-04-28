package objects

import "time"

// objects stores information to contact with the pf9 controller.
/*type Config struct {
	Fqdn     string `json:"fqdn"`
	Username string `json:"username"`
	Password string `json:"password"`
	Tenant   string `json:"tenant"`
	Region   string `json:"region"`
	ProxyURL string `json:"proxy_url"`
	MfaToken string `json:"mfa_token"`
}*/

/*type NodeConfig struct {
	User     string
	Password string
	SshKey   string
	IPs      []string
}*/

type NodeConfig struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Spec       Spec   `yaml:"spec"`
	SshKey     string `yaml:"ssh-key"`
	Password   string `yaml:"password,omitempty" `
}
type Node struct {
	Ip       string `yaml:"ip"`
	Hostname string `yaml:"hostname"`
	Type     string `yaml:"type"`
}
type Spec struct {
	DeploymentKind string `yaml:"deploymentKind"`
	Type           string `yaml:"type"`
	Nodes          []Node `yaml:"nodes"`
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

/*type ClusterConfig struct {
	Name                      string     `json:"name"`
	MasterNodes               []string   `json:"masterNodes"`
	AllowWorkloadsOnMaster    bool       `json:"allowWorkloadsOnMaster"`
	WorkerNodes               []string   `json:"workerNodes"`
	ContainersCidr            string     `json:"containersCidr"`
	ServicesCidr              string     `json:"servicesCidr"`
	MtuSize                   int        `json:"mtuSize"`
	Privileged                bool       `json:"privileged"`
	DeployLuigiOperator       bool       `json:"deployLuigiOperator"`
	UseHostname               bool       `json:"useHostname"`
	NodePoolUUID              string     `json:"nodePoolUuid"`
	EnableProfileAgent        bool       `json:"enableProfileAgent"`
	KubeRoleVersion           string     `json:"kubeRoleVersion"`
	CalicoIPIPMode            string     `json:"calicoIpIpMode"`
	CalicoNatOutgoing         bool       `json:"calicoNatOutgoing"`
	CalicoV4BlockSize         string     `json:"calicoV4BlockSize"`
	CalicoIPv4DetectionMethod string     `json:"calicoIPv4DetectionMethod"`
	NetworkPlugin             string     `json:"networkPlugin"`
	RuntimeConfig             string     `json:"runtimeConfig"`
	ContainerRuntime          string     `json:"containerRuntime"`
	EtcdBackup                EtcdBackup `json:"etcdBackup"`
	Monitoring                Monitoring `json:"monitoring"`
	Tags                      Tags       `json:"tags"`
}
type StorageProperties struct {
	LocalPath string `json:"localPath"`
}
type EtcdBackup struct {
	StorageType             string            `json:"storageType"`
	IsEtcdBackupEnabled     int               `json:"isEtcdBackupEnabled"`
	StorageProperties       StorageProperties `json:"storageProperties"`
	DailyBackupTime         string            `json:"dailyBackupTime"`
	MaxTimestampBackupCount int               `json:"maxTimestampBackupCount"`
	IntervalInHours         int               `json:"intervalInHours"`
	MaxIntervalBackupCount  int               `json:"maxIntervalBackupCount"`
}
type Monitoring struct {
	RetentionTime string `json:"retentionTime"`
}
type Tags struct {
}*/
