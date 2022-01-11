package config

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/platform9/pf9ctl/pkg/client"
	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/keystone"
	"github.com/platform9/pf9ctl/pkg/objects"

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
	cfgCopy.Password = base64.StdEncoding.EncodeToString([]byte(cfg.Password))

	// Clear the MFA token as it will be required afresh every time
	cfgCopy.MfaToken = ""

	f, err := os.Create(loc)
	if err != nil {
		return err
	}

	defer f.Close()

	encoder := json.NewEncoder(f)
	fmt.Println(color.Green("✓ ") + "Stored configuration details Succesfully")
	return encoder.Encode(cfgCopy)
}

// LoadConfig returns the information for communication with PF9 controller.
func LoadConfig(loc string, cfg *objects.Config, nc objects.NodeConfig) error {

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
	err = json.NewDecoder(f).Decode(&fileConfig)
	// decode the password
	// Decoding base64 encoded password
	decodedBytePassword, err := base64.StdEncoding.DecodeString(fileConfig.Password)
	if err != nil {
		return err
	}
	fileConfig.Password = string(decodedBytePassword)
	//s.Stop()

	copier.CopyWithOption(cfg, &fileConfig, copier.Option{IgnoreEmpty: true})

	if err = SetProxy(cfg.ProxyURL); err != nil {
		return err
	}

	return ValidateUserCredentials(cfg, nc)
}

func LoadConfigInteractive(loc string, cfg *objects.Config, nc objects.NodeConfig) error {

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

func GetConfigRecursive(loc string, cfg *objects.Config, nc objects.NodeConfig) error {
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

func ValidateUserCredentials(cfg *objects.Config, nc objects.NodeConfig) error {

	if err := validateConfigFields(cfg); err != nil {
		return err
	}

	c, err := createClient(cfg, nc)
	defer c.Segment.Close()
	if err != nil {
		return fmt.Errorf("Error validating credentials %w", err)
	}

	auth, err := c.Keystone.GetAuth(
		cfg.Username,
		cfg.Password,
		cfg.Tenant,
		cfg.MfaToken,
	)
	if err != nil {
		zap.S().Debug(err)
		return INVALID_CREDS
	}

	// To validate region.
	endpointURL, err1 := keystone.FetchRegionFQDN(cfg.Fqdn, cfg.Region, auth)
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

	if cfg.AwsIamUsername == "" {
		fmt.Printf("Amazon IAM User: ")
		awsIamUsername, _ := reader.ReadString('\n')
		cfg.AwsIamUsername = strings.TrimSuffix(awsIamUsername, "\n")
	}

	if cfg.AwsAccessKey == "" {
		fmt.Printf("Amazon Access Key: ")
		accessKey, _ := terminal.ReadPassword(0)
		cfg.AwsAccessKey = string(accessKey)
		fmt.Println()
	}

	if cfg.AwsSecretKey == "" {
		fmt.Printf("Amazon Secret Key: ")
		secretKey, _ := terminal.ReadPassword(0)
		cfg.AwsSecretKey = string(secretKey)
		fmt.Println()
	}
	var region string
	if cfg.AwsRegion == "" {
		fmt.Printf("Region: ")
		region, _ = reader.ReadString('\n')
		cfg.AwsRegion = strings.TrimSuffix(region, "\n")
	}

	if cfg.AwsRegion == "" {
		cfg.AwsRegion = "us-east-1"
	}

	return nil
}

func ConfigCmdCreateAzureRun(cfg *objects.Config) error {

	zap.S().Debug("==========Running set config==========")

	if cfg.AzureTenant == "" {
		fmt.Printf("Azure TenantID: ")
		azureTenant, _ := terminal.ReadPassword(0)
		cfg.AzureTenant = string(azureTenant)
		fmt.Println()
	}

	if cfg.AzureClient == "" {
		fmt.Printf("Azure ApplicationID: ")
		azureClient, _ := terminal.ReadPassword(0)
		cfg.AzureClient = string(azureClient)
		fmt.Println()
	}

	if cfg.AzureSubscription == "" {
		fmt.Printf("Azure SubscriptionID: ")
		azureSub, _ := terminal.ReadPassword(0)
		cfg.AzureSubscription = string(azureSub)
		fmt.Println()
	}

	if cfg.AzureSecret == "" {
		fmt.Printf("\nAzure Secret Key: ")
		azureSecret, _ := terminal.ReadPassword(0)
		cfg.AzureSecret = string(azureSecret)
		fmt.Println()
	}

	return nil
}

func ConfigCmdCreateGoogleRun(cfg *objects.Config) error {

	zap.S().Debug("==========Running set config==========")

	reader := bufio.NewReader(os.Stdin)

	if cfg.GooglePath == "" {
		fmt.Printf("Service JSON path: ")
		googleProjectName, _ := reader.ReadString('\n')
		cfg.GoogleProjectName = strings.TrimSuffix(googleProjectName, "\n")
	}

	if cfg.GoogleProjectName == "" {
		fmt.Printf("Project Name: ")
		googleProjectName, _ := reader.ReadString('\n')
		cfg.GoogleProjectName = strings.TrimSuffix(googleProjectName, "\n")
	}

	if cfg.GoogleServiceEmail == "" {
		fmt.Printf("Service Account Email: ")
		googleServiceEmail, _ := reader.ReadString('\n')
		cfg.GoogleServiceEmail = strings.TrimSuffix(googleServiceEmail, "\n")
	}

	return nil
}

// ConfigCmdCreatRun will initiate the config set and return a config given by user
func ConfigCmdCreateRun(cfg *objects.Config) error {

	zap.S().Debug("==========Running set config==========")

	reader := bufio.NewReader(os.Stdin)

	if cfg.Fqdn == "" {
		fmt.Printf("Platform9 Account URL: ")
		fqdn, _ := reader.ReadString('\n')
		cfg.Fqdn = strings.TrimSuffix(fqdn, "\n")
	}

	if cfg.Username == "" {
		fmt.Printf("Username: ")
		username, _ := reader.ReadString('\n')
		cfg.Username = strings.TrimSuffix(username, "\n")
	}

	if cfg.Password == "" {
		fmt.Printf("Password: ")
		passwordBytes, _ := terminal.ReadPassword(0)
		cfg.Password = string(passwordBytes)
		fmt.Println()
	}
	var region string
	if cfg.Region == "" {
		fmt.Printf("Region [RegionOne]: ")
		region, _ = reader.ReadString('\n')
		cfg.Region = strings.TrimSuffix(region, "\n")
	}
	var service string
	if cfg.Tenant == "" {
		fmt.Printf("Tenant [service]: ")
		service, _ = reader.ReadString('\n')
		cfg.Tenant = strings.TrimSuffix(service, "\n")
	}
	var proxyURL string
	if cfg.ProxyURL == "" {
		fmt.Print("Proxy URL [None]: ")
		proxyURL, _ = reader.ReadString('\n')
		cfg.ProxyURL = strings.TrimSuffix(proxyURL, "\n")
	}

	if cfg.NoProxy == "" {
		fmt.Print("Noproxy [None]: ")
		URL, _ := reader.ReadString('\n')
		cfg.NoProxy = strings.TrimSuffix(URL, "\n")
	}

	if cfg.Region == "" {
		cfg.Region = "RegionOne"
	}

	if cfg.Tenant == "" {
		cfg.Tenant = "service"
	}

	var mfaToken string
	if cfg.MfaToken == "" {
		fmt.Print("MFA Token [None]: ")
		mfaToken, _ = reader.ReadString('\n')
		cfg.MfaToken = strings.TrimSuffix(mfaToken, "\n")
	}

	return SetProxy(cfg.ProxyURL)
}

func createClient(cfg *objects.Config, nc objects.NodeConfig) (client.Client, error) {
	executor, err := cmdexec.GetExecutor(cfg.ProxyURL, cfg.NoProxy, nc)
	if err != nil {
		//debug first since Fatalf calls os.Exit
		zap.S().Debug("Error connecting to host %s", err.Error())
		zap.S().Fatalf(" Invalid (Username/Password/IP), use 'single quotes' to pass password")
	}

	return client.NewClient(cfg.Fqdn, executor, cfg.AllowInsecure, false)
}

//This function clears the context if it is invalid. Before storing it.
func clearContext(v interface{}) {
	p := reflect.ValueOf(v).Elem()
	p.Set(reflect.Zero(p.Type()))
}

func ValidateNodeConfig(nc *objects.NodeConfig, interactive bool) bool {

	if nc.User == "" || (nc.SshKey == "" && nc.Password == "") {
		if !interactive {
			return false
		}

		if nc.User == "" {
			fmt.Printf("Enter username for remote host: ")
			reader := bufio.NewReader(os.Stdin)
			nc.User, _ = reader.ReadString('\n')
			nc.User = strings.TrimSpace(nc.User)
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

func SetNoProxy(NoProxy string) error {
	if NoProxy != "" {
		if err := os.Setenv("no_proxy", NoProxy); err != nil {
			return errors.New("Error setting no_proxy as environment variable")
		}
	}
	return nil
}

func validateConfigFields(cfg *objects.Config) error {
	if cfg.Fqdn == "" || cfg.Username == "" || cfg.Password == "" || cfg.Region == "" || cfg.Tenant == "" {
		return MISSSING_FIELDS
	}
	return nil
}
