package config

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/platform9/pf9ctl/pkg/client"
	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/keystone"
	"github.com/platform9/pf9ctl/pkg/objects"
	"gopkg.in/yaml.v2"

	"github.com/jinzhu/copier"
	"github.com/platform9/pf9ctl/pkg/util"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	REGION_INVALID     error = errors.New("Invalid Region")
	INVALID_CREDS      error = errors.New("Invalid Credentials")
	NO_CONFIG                = errors.New("No config found, please create with `pf9ctl config set`")
	MISSSING_FIELDS          = errors.New("Missing mandatory field(s) (Platform9 Account URL/Username/Password/Region/Tenant)")
	MAX_ATTEMPTS_ERROR       = errors.New("Invalid credentials entered multiple times (Platform9 Account URL/Username/Password/Region/Tenant/Proxy URL/MFA Token)")
)

// StoreConfig simply updates the in-memory object
func StoreConfig(cfg *objects.Config, loc string) error {
	zap.S().Debug("Storing configuration details")

	var cfgCopy objects.Config

	copier.CopyWithOption(&cfgCopy, cfg, copier.Option{IgnoreEmpty: true, DeepCopy: true})

	// obscure the password
	cfgCopy.Spec.Password = base64.StdEncoding.EncodeToString([]byte(cfg.Spec.Password))

	// Clear the MFA token as it will be required afresh every time
	cfgCopy.Spec.MfaToken = ""

	f, err := os.Create(loc)
	if err != nil {
		return err
	}

	defer f.Close()

	encoder := json.NewEncoder(f)
	fmt.Println(color.Green("✓ ") + "Stored configuration details successfully")
	return encoder.Encode(cfgCopy)
}

// LoadConfig returns the information for communication with PF9 controller.
func LoadConfig(loc string, cfg *objects.Config, nc *objects.NodeConfig) error {

	zap.S().Debug("Loading configuration details. pf9ctl version: ", util.Version)

	f, err := os.Open(loc)
	if err != nil {
		if os.IsNotExist(err) {
			zap.S().Debug(NO_CONFIG.Error())
			return NO_CONFIG
		} else {
			zap.S().Debug(err.Error())
			return err
		}
	}

	defer f.Close()

	var fileConfig objects.Config

	ext := filepath.Ext(loc)
	if ext != ".yaml" {
		err = json.NewDecoder(f).Decode(&fileConfig)
	} else {
		err = yaml.NewDecoder(f).Decode(&fileConfig)
	}
	// decode the password
	// Decoding base64 encoded password
	decodedBytePassword, err := base64.StdEncoding.DecodeString(fileConfig.Spec.Password)
	if err != nil {
		return err
	}
	fileConfig.Spec.Password = string(decodedBytePassword)
	//s.Stop()

	copier.CopyWithOption(cfg, &fileConfig, copier.Option{IgnoreEmpty: true})

	if err = SetProxy(cfg.Spec.ProxyURL); err != nil {
		return err
	}

	return ValidateUserCredentials(cfg, nc)
}

func LoadConfigInteractive(loc string, cfg *objects.Config, nc *objects.NodeConfig) error {

	err := LoadConfig(loc, cfg, nc)
	if err == nil {
		return nil
	}

	if err == NO_CONFIG {
		fmt.Println(color.Red("x ") + "Existing config not found, prompting for new config")
		zap.S().Debug("Existing config not found, prompting for new config.")
	} else if err == INVALID_CREDS || err == REGION_INVALID {
		fmt.Println(color.Red("x ") + "Existing config is invalid, prompting for new config")
		zap.S().Debug("Existing config is invalid, prompting for new config.")
	}

	clearContext(cfg)
	return GetConfigRecursive(loc, cfg, nc)

}

func GetConfigRecursive(loc string, cfg *objects.Config, nc *objects.NodeConfig) error {
	maxLoopNoConfig := 3
	InvalidExistingConfig := false
	count := 0
	var err error

	if loc == "amazon.json" {
		return ConfigCmdCreateAmazonRun(cfg)
	} else if loc == "azure.json" {
		return ConfigCmdCreateAzureRun(cfg)
	} else if loc == "google.json" {
		return ConfigCmdCreateGoogleRun(cfg)
	}

	for count < maxLoopNoConfig {

		if InvalidExistingConfig {
			fmt.Println(color.Red("x ") + "Invalid credentials entered (Platform9 Account URL/Username/Password/Region/Tenant/MFA Token)")
			zap.S().Debug("Invalid config entered, prompting for new config.")
		}

		err = ConfigCmdCreateRun(cfg)

		if err != nil {
			fmt.Println("\n" + color.Red("x ") + err.Error())
			clearContext(cfg)
			InvalidExistingConfig = true
			count++
			continue
		}

		if err = ValidateUserCredentials(cfg, nc); err != nil {
			clearContext(cfg)
			InvalidExistingConfig = true
			count++
			continue
		}

		return StoreConfig(cfg, util.Pf9DBLoc)
	}

	if InvalidExistingConfig && count == maxLoopNoConfig {
		return MAX_ATTEMPTS_ERROR
	}
	return err
}

func ValidateUserCredentials(cfg *objects.Config, nc *objects.NodeConfig) error {

	if err := validateConfigFields(cfg); err != nil {
		return err
	}

	c, err := createClient(cfg, nc)
	defer c.Segment.Close()
	if err != nil {
		return fmt.Errorf("Error validating credentials %w", err)
	}

	auth, err := c.Keystone.GetAuth(
		cfg.Spec.Username,
		cfg.Spec.Password,
		cfg.Spec.Tenant,
		cfg.Spec.MfaToken,
	)
	if err != nil {
		zap.S().Debug(err)
		return INVALID_CREDS
	}

	// To validate region.
	endpointURL, err1 := keystone.FetchRegionFQDN(cfg.Spec.AccountUrl, cfg.Spec.Region, auth)
	if endpointURL == "" || err1 != nil {
		zap.S().Debug("Invalid Region")
		return REGION_INVALID
	}
	return nil
}

// var cfg objects.Config

func ConfigCmdCreateAmazonRun(cfg *objects.Config) error {

	zap.S().Debug("==========Running set config==========")

	reader := bufio.NewReader(os.Stdin)

	if cfg.Spec.AWS.AwsIamUsername == "" {
		fmt.Printf("Amazon IAM User: ")
		awsIamUsername, _ := reader.ReadString('\n')
		cfg.Spec.AWS.AwsIamUsername = strings.TrimSuffix(awsIamUsername, "\n")
	}

	if cfg.Spec.AWS.AwsAccessKey == "" {
		fmt.Printf("Amazon Access Key: ")
		accessKey, _ := terminal.ReadPassword(0)
		cfg.Spec.AWS.AwsAccessKey = string(accessKey)
		fmt.Println()
	}

	if cfg.Spec.AWS.AwsSecretKey == "" {
		fmt.Printf("Amazon Secret Key: ")
		secretKey, _ := terminal.ReadPassword(0)
		cfg.Spec.AWS.AwsSecretKey = string(secretKey)
		fmt.Println()
	}
	var region string
	if cfg.Spec.AWS.AwsRegion == "" {
		fmt.Printf("Region: ")
		region, _ = reader.ReadString('\n')
		cfg.Spec.AWS.AwsRegion = strings.TrimSuffix(region, "\n")
	}

	if cfg.Spec.AWS.AwsRegion == "" {
		cfg.Spec.AWS.AwsRegion = "us-east-1"
	}

	return nil
}

func ConfigCmdCreateAzureRun(cfg *objects.Config) error {

	zap.S().Debug("==========Running set config==========")

	if cfg.Spec.Azure.AzureTenant == "" {
		fmt.Printf("Azure TenantID: ")
		azureTenant, _ := terminal.ReadPassword(0)
		cfg.Spec.Azure.AzureTenant = string(azureTenant)
		fmt.Println()
	}

	if cfg.Spec.Azure.AzureClient == "" {
		fmt.Printf("Azure ApplicationID: ")
		azureClient, _ := terminal.ReadPassword(0)
		cfg.Spec.Azure.AzureClient = string(azureClient)
		fmt.Println()
	}

	if cfg.Spec.Azure.AzureSubscription == "" {
		fmt.Printf("Azure SubscriptionID: ")
		azureSub, _ := terminal.ReadPassword(0)
		cfg.Spec.Azure.AzureSubscription = string(azureSub)
		fmt.Println()
	}

	if cfg.Spec.Azure.AzureSecret == "" {
		fmt.Printf("\nAzure Secret Key: ")
		azureSecret, _ := terminal.ReadPassword(0)
		cfg.Spec.Azure.AzureSecret = string(azureSecret)
		fmt.Println()
	}

	return nil
}

func ConfigCmdCreateGoogleRun(cfg *objects.Config) error {

	zap.S().Debug("==========Running set config==========")

	reader := bufio.NewReader(os.Stdin)

	if cfg.Spec.Google.GooglePath == "" {
		fmt.Printf("Service JSON path: ")
		googleProjectName, _ := reader.ReadString('\n')
		cfg.Spec.Google.GoogleProjectName = strings.TrimSuffix(googleProjectName, "\n")
	}

	if cfg.Spec.Google.GoogleProjectName == "" {
		fmt.Printf("Project Name: ")
		googleProjectName, _ := reader.ReadString('\n')
		cfg.Spec.Google.GoogleProjectName = strings.TrimSuffix(googleProjectName, "\n")
	}

	if cfg.Spec.Google.GoogleServiceEmail == "" {
		fmt.Printf("Service Account Email: ")
		googleServiceEmail, _ := reader.ReadString('\n')
		cfg.Spec.Google.GoogleServiceEmail = strings.TrimSuffix(googleServiceEmail, "\n")
	}

	return nil
}

// ConfigCmdCreatRun will initiate the config set and return a config given by user
func ConfigCmdCreateRun(cfg *objects.Config) error {

	zap.S().Debug("==========Running set config==========")

	reader := bufio.NewReader(os.Stdin)

	if cfg.Spec.AccountUrl == "" {
		fmt.Printf("Platform9 Account URL: ")
		fqdn, _ := reader.ReadString('\n')
		cfg.Spec.AccountUrl = strings.TrimSuffix(fqdn, "\n")
	}

	if cfg.Spec.Username == "" {
		fmt.Printf("Username: ")
		username, _ := reader.ReadString('\n')
		cfg.Spec.Username = strings.TrimSuffix(username, "\n")
	}

	if cfg.Spec.Password == "" {
		fmt.Printf("Password: ")
		passwordBytes, _ := terminal.ReadPassword(0)
		cfg.Spec.Password = string(passwordBytes)
		fmt.Println()
	}
	var region string
	if cfg.Spec.Region == "" {
		fmt.Printf("Region [RegionOne]: ")
		region, _ = reader.ReadString('\n')
		cfg.Spec.Region = strings.TrimSuffix(region, "\n")
	}
	var service string
	if cfg.Spec.Tenant == "" {
		fmt.Printf("Tenant [service]: ")
		service, _ = reader.ReadString('\n')
		cfg.Spec.Tenant = strings.TrimSuffix(service, "\n")
	}
	var proxyURL string
	if cfg.Spec.ProxyURL == "" {
		fmt.Print("Proxy URL [None]: ")
		proxyURL, _ = reader.ReadString('\n')
		cfg.Spec.ProxyURL = strings.TrimSuffix(proxyURL, "\n")
	}

	if cfg.Spec.Region == "" {
		cfg.Spec.Region = "RegionOne"
	}

	if cfg.Spec.Tenant == "" {
		cfg.Spec.Tenant = "service"
	}

	var mfaToken string
	if cfg.Spec.MfaToken == "" {
		fmt.Print("MFA Token [None]: ")
		mfaToken, _ = reader.ReadString('\n')
		cfg.Spec.MfaToken = strings.TrimSuffix(mfaToken, "\n")
	}

	return SetProxy(cfg.Spec.ProxyURL)
}

func createClient(cfg *objects.Config, nc *objects.NodeConfig) (client.Client, error) {
	executor, err := cmdexec.GetExecutor(cfg.Spec.ProxyURL, nc)
	if err != nil {
		//debug first since Fatalf calls os.Exit
		zap.S().Debug("Error connecting to host %s", err.Error())
		zap.S().Fatalf(" Invalid (Username/Password/IP), use 'single quotes' to pass password")
	}

	return client.NewClient(cfg.Spec.AccountUrl, executor, cfg.Spec.OtherData.AllowInsecure, false)
}

//This function clears the context if it is invalid. Before storing it.
func clearContext(v interface{}) {
	p := reflect.ValueOf(v).Elem()
	p.Set(reflect.Zero(p.Type()))
}

func ValidateNodeConfig(nc *objects.NodeConfig, interactive bool) bool {

	if nc.Spec.Nodes[0].Hostname == "" || (nc.SshKey == "" && nc.Password == "") {
		if !interactive {
			return false
		}

		if nc.Spec.Nodes[0].Hostname == "" {
			fmt.Printf("Enter username for remote host: ")
			reader := bufio.NewReader(os.Stdin)
			nc.Spec.Nodes[0].Hostname, _ = reader.ReadString('\n')
			nc.Spec.Nodes[0].Hostname = strings.TrimSpace(nc.Spec.Nodes[0].Hostname)
		}
		if nc.SshKey == "" && nc.Password == "" {
			var choice int
			fmt.Println("You can choose either password or sshKey")
			fmt.Println("Enter 1 for password and 2 for sshKey")
			fmt.Print("Enter Option : ")
			fmt.Scanf("%d", &choice)
			switch choice {
			case 1:
				fmt.Printf("Enter password for remote host: ")
				passwordBytes, _ := terminal.ReadPassword(0)
				nc.Password = string(passwordBytes)
			case 2:
				fmt.Printf("Enter private SSH key: ")
				reader := bufio.NewReader(os.Stdin)
				nc.SshKey, _ = reader.ReadString('\n')
				nc.SshKey = strings.TrimSpace(nc.SshKey)
			default:
				zap.S().Fatalf("Wrong choice please try again")
			}
			fmt.Printf("\n")
		}
	}
	return true
}

func SetProxy(proxyURL string) error {
	if proxyURL != "" {
		if err := os.Setenv("https_proxy", proxyURL); err != nil {
			return errors.New("Error setting proxy as environment variable")
		}
	}
	return nil
}

func validateConfigFields(cfg *objects.Config) error {
	if cfg.Spec.AccountUrl == "" || cfg.Spec.Username == "" || cfg.Spec.Password == "" || cfg.Spec.Region == "" || cfg.Spec.Tenant == "" {
		return MISSSING_FIELDS
	}
	return nil
}

func LoadNodeConfig(nc *objects.NodeConfig, loc string) {
	f, err := os.Open(loc)
	if err != nil {
		fmt.Println("Error opening node config file")
	}
	defer f.Close()
	var nodec objects.NodeConfig
	ext := filepath.Ext(loc)
	if ext != ".yaml" {
		err = json.NewDecoder(f).Decode(&nodec)
	} else {
		err = yaml.NewDecoder(f).Decode(&nodec)
	}
	copier.CopyWithOption(nc, &nodec, copier.Option{IgnoreEmpty: true})
}
