package objects

import "time"

// objects stores information to contact with the pf9 controller.
type Config struct {
	Fqdn               string        `json:"fqdn"`
	Username           string        `json:"username"`
	Password           string        `json:"password"`
	Tenant             string        `json:"tenant"`
	Region             string        `json:"region"`
	WaitPeriod         time.Duration `json:"wait_period"`
	AllowInsecure      bool          `json:"allow_insecure"`
	ProxyURL           string        `json:"proxy_url"`
	MfaToken           string        `json:"mfa_token"`
	AwsIamUsername     string        `json:"aws_iam_username"`
	AwsAccessKey       string        `json:"aws_access_key"`
	AwsSecretKey       string        `json:"aws_secret_key"`
	AwsRegion          string        `json:"aws_region"`
	AzureTenant        string        `json:"azure_tenant"`
	AzureClient        string        `json:"azure_application"`
	AzureSubscription  string        `json:"azure_subscription"`
	AzureSecret        string        `json:"azure_secret"`
	GooglePath         string        `json:"google_path"`
	GoogleProjectName  string        `json:"google_project_name"`
	GoogleServiceEmail string        `json:"google_service_email"`
}

type NodeConfig struct {
	User         string
	Password     string
	SshKey       string
	IPs          []string
	MFA          string
	SudoPassword string
}
