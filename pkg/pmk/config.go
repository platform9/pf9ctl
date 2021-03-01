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

	//"github.com/platform9/pf9ctl/pkg/util"

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh/terminal"
)

var ErrConfigurationDetailsNotProvided = errors.New("config not set,....")

var (
	allClients Client
	// This flag is set true when we set confing for first time
	IfNewConfig = false
	ctx         Config
)

// Config stores information to contact with the pf9 controller.
type Config struct {
	Fqdn          string        `json:"fqdn"`
	Username      string        `json:"username"`
	Password      string        `json:"password"`
	Tenant        string        `json:"tenant"`
	Region        string        `json:"region"`
	WaitPeriod    time.Duration `json:"wait_period"`
	AllowInsecure bool          `json:"allow_insecure"`
}

// StoreConfig simply updates the in-memory object
func StoreConfig(ctx Config, loc string) error {
	zap.S().Info("Storing configuration details")
	// obscure the password
	ctx.Password = base64.StdEncoding.EncodeToString([]byte(ctx.Password))
	f, err := os.Create(loc)
	if err != nil {
		return err
	}
	defer f.Close()
	encoder := json.NewEncoder(f)
	return encoder.Encode(ctx)

}

// LoadConfig returns the information for communication with PF9 controller.
func LoadConfig(loc string) (Config, error) {
	zap.S().Info("Loading configuration details")

	f, err := os.Open(loc)
	if err != nil {

		if os.IsNotExist(err) {
			// to initiate the config create and store it
			zap.S().Info("Existing config not found, prompting for new config.")
			ctx, err = ConfigCmdCreateRun()

			return ctx, err
		}
		return Config{}, err
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

	return ctx, err
}

// ConfigCmdCreatRun will initiate the config set and return a config given by user
func ConfigCmdCreateRun() (Config, error) {

	zap.S().Info("==========Running set config==========")

	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("Platform9 Account URL: ")
	fqdn, _ := reader.ReadString('\n')
	fqdn = strings.TrimSuffix(fqdn, "\n")

	fmt.Printf("Username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSuffix(username, "\n")

	fmt.Printf("Password: ")
	passwordBytes, _ := terminal.ReadPassword(0)
	password := string(passwordBytes)

	fmt.Printf("\nRegion [RegionOne]: ")
	region, _ := reader.ReadString('\n')
	region = strings.TrimSuffix(region, "\n")

	fmt.Printf("Tenant [service]: ")
	service, _ := reader.ReadString('\n')
	service = strings.TrimSuffix(service, "\n")

	if region == "" {
		region = "RegionOne"
	}

	if service == "" {
		service = "service"
	}

	ctx := Config{
		Fqdn:          fqdn,
		Username:      username,
		Password:      password,
		Region:        region,
		Tenant:        service,
		WaitPeriod:    time.Duration(60),
		AllowInsecure: false,
	}
	IfNewConfig = true
	return ctx, nil
}
