package pmk

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/util"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	IsNewConfig           bool
	OldConfigExist        bool
	LoopCounter           int
	InvalidExistingConfig bool
)

// Config stores information to contact with the pf9 controller.
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
	AzureTetant        string        `json:"azure_tenant"`
	AzureApplication   string        `json:"azure_application"`
	AzureSubscription  string        `json:"azure_subscription"`
	AzureSecret        string        `json:"azure_secret"`
	GoogleProjectName  string        `json:"google_project_name"`
	GoogleServiceEmail string        `json:"google_service_email"`
}

// StoreConfig simply updates the in-memory object
func StoreConfig(ctx Config, loc string) error {
	zap.S().Debug("Storing configuration details")

	// obscure the password
	ctx.Password = base64.StdEncoding.EncodeToString([]byte(ctx.Password))

	// Clear the MFA token as it will be required afresh every time
	ctx.MfaToken = ""

	f, err := os.Create(loc)
	if err != nil {
		return err
	}

	defer f.Close()

	encoder := json.NewEncoder(f)
	fmt.Println(color.Green("✓ ") + "Stored configuration details Succesfully")
	return encoder.Encode(ctx)

}

// LoadConfig returns the information for communication with PF9 controller.
func LoadConfig(loc string) (Config, error) {

	zap.S().Debug("Loading configuration details. pf9ctl version: ", util.Version)

	f, err := os.Open(loc)
	// We will execute it if no config found or if config found but have invalid credentials
	if err != nil || (err == nil && InvalidExistingConfig) {

		if os.IsNotExist(err) || InvalidExistingConfig {
			// If Config not found and we prompt for new config
			if LoopCounter == 0 {
				fmt.Println(color.Red("x ") + "Existing config not found, prompting for new config")
				zap.S().Debug("Existing config not found, prompting for new config.")
				// to initiate the config create and store it
			}
			// If Existing config is invalid then we prompt for new config
			if InvalidExistingConfig && LoopCounter == 1 {
				fmt.Println(color.Red("x ") + "Existing config is invalid, prompting for new config")
				zap.S().Debug("Existing config is invalid, prompting for new config.")
			}

			IsNewConfig = true

			if loc == "amazon.json" {
				return ConfigCmdCreateAmazonRun()
			} else if loc == "azure.json" {
				return ConfigCmdCreateAzureRun()
			} else if loc == "google.json" {
				return ConfigCmdCreateGoogleRun()
			} else {
				return ConfigCmdCreateRun()
			}

			// It is set true when we are setting config for the first time using check-node/prep-node

		}
		return Config{}, err
	}
	if LoopCounter == 0 {
		OldConfigExist = true
	}
	defer f.Close()

	ctx := Config{WaitPeriod: time.Duration(60), AllowInsecure: false}
	err = json.NewDecoder(f).Decode(&ctx)
	// decode the password
	// Decoding base64 encoded password
	decodedBytePassword, err := base64.StdEncoding.DecodeString(ctx.Password)
	if err != nil {
		return ctx, err
	}
	ctx.Password = string(decodedBytePassword)
	//s.Stop()
	fmt.Println(color.Green("✓ ") + "Loaded Config Successfully")

	if ctx.ProxyURL != "" {
		if err = os.Setenv("https_proxy", ctx.ProxyURL); err != nil {
			return Config{}, errors.New("Error setting proxy as environment variable")
		}
	}

	return ctx, err
}

var Context Config

func ConfigCmdCreateAmazonRun() (Config, error) {

	zap.S().Debug("==========Running set config==========")

	reader := bufio.NewReader(os.Stdin)

	if Context.AwsIamUsername == "" {
		fmt.Printf("Amazon IAM User: ")
		awsIamUsername, _ := reader.ReadString('\n')
		Context.AwsIamUsername = strings.TrimSuffix(awsIamUsername, "\n")
	}

	if Context.AwsAccessKey == "" {
		fmt.Printf("Amazon Access Key: ")
		accessKey, _ := terminal.ReadPassword(0)
		Context.AwsAccessKey = string(accessKey)
	}

	if Context.AwsSecretKey == "" {
		fmt.Printf("\nAmazon Secret Key: ")
		secretKey, _ := terminal.ReadPassword(0)
		Context.AwsSecretKey = string(secretKey)
	}
	var region string
	if Context.AwsRegion == "" {
		fmt.Printf("\nRegion [us-east-1]: ")
		region, _ = reader.ReadString('\n')
		Context.AwsRegion = strings.TrimSuffix(region, "\n")
	}

	if Context.AwsRegion == "" {
		Context.AwsRegion = "us-east-1"
	}

	return Context, nil

}

func ConfigCmdCreateAzureRun() (Config, error) {

	zap.S().Debug("==========Running set config==========")

	if Context.AzureTetant == "" {
		fmt.Printf("Azure TenantID: ")
		azureTenant, _ := terminal.ReadPassword(0)
		Context.AzureTetant = string(azureTenant)
	}

	if Context.AzureApplication == "" {
		fmt.Printf("\nAzure ApplicationID: ")
		azureApp, _ := terminal.ReadPassword(0)
		Context.AzureApplication = string(azureApp)
	}

	if Context.AzureSubscription == "" {
		fmt.Printf("\nAzure SubscriptionID: ")
		azureSub, _ := terminal.ReadPassword(0)
		Context.AzureSubscription = string(azureSub)
	}

	if Context.AzureSecret == "" {
		fmt.Printf("\nAzure Secret Key: ")
		azureSecret, _ := terminal.ReadPassword(0)
		Context.AzureSecret = string(azureSecret)
	}

	fmt.Printf("\n")

	return Context, nil

}

func ConfigCmdCreateGoogleRun() (Config, error) {

	zap.S().Debug("==========Running set config==========")

	reader := bufio.NewReader(os.Stdin)

	if Context.AwsIamUsername == "" {
		fmt.Printf("Project Name: ")
		googleProjectName, _ := reader.ReadString('\n')
		Context.GoogleProjectName = strings.TrimSuffix(googleProjectName, "\n")
	}

	if Context.AwsIamUsername == "" {
		fmt.Printf("Service Account Email: ")
		googleServiceEmail, _ := reader.ReadString('\n')
		Context.GoogleServiceEmail = strings.TrimSuffix(googleServiceEmail, "\n")
	}

	return Context, nil

}

// ConfigCmdCreatRun will initiate the config set and return a config given by user
func ConfigCmdCreateRun() (Config, error) {

	zap.S().Debug("==========Running set config==========")

	reader := bufio.NewReader(os.Stdin)

	if Context.Fqdn == "" {
		fmt.Printf("Platform9 Account URL: ")
		fqdn, _ := reader.ReadString('\n')
		Context.Fqdn = strings.TrimSuffix(fqdn, "\n")
	}

	if Context.Username == "" {
		fmt.Printf("Username: ")
		username, _ := reader.ReadString('\n')
		Context.Username = strings.TrimSuffix(username, "\n")
	}

	if Context.Password == "" {
		fmt.Printf("Password: ")
		passwordBytes, _ := terminal.ReadPassword(0)
		Context.Password = string(passwordBytes)
	}
	var region string
	if Context.Region == "" {
		fmt.Printf("\nRegion [RegionOne]: ")
		region, _ = reader.ReadString('\n')
		Context.Region = strings.TrimSuffix(region, "\n")
	}
	var service string
	if Context.Tenant == "" {
		fmt.Printf("Tenant [service]: ")
		service, _ = reader.ReadString('\n')
		Context.Tenant = strings.TrimSuffix(service, "\n")
	}
	var proxyURL string
	if Context.ProxyURL == "" {
		fmt.Print("Proxy URL [None]: ")
		proxyURL, _ = reader.ReadString('\n')
		Context.ProxyURL = strings.TrimSuffix(proxyURL, "\n")
	}

	if Context.Region == "" {
		Context.Region = "RegionOne"
	}

	if Context.Tenant == "" {
		Context.Tenant = "service"
	}

	var mfaToken string
	if Context.MfaToken == "" {
		fmt.Print("MFA Token [None]: ")
		mfaToken, _ = reader.ReadString('\n')
		Context.MfaToken = strings.TrimSuffix(mfaToken, "\n")
	}

	return Context, nil

}
