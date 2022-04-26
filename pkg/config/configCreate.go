package config

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/platform9/pf9ctl/pkg/objects"
	"github.com/platform9/pf9ctl/pkg/util"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/yaml.v2"
)

var (
	cfg          objects.Config
	JsonFileType bool
	FileName     string
	Location     string
)

func CreateUserConfig() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("API version: ")
	apiVersion, _ := reader.ReadString('\n')
	cfg.ApiVersion = strings.TrimSuffix(apiVersion, "\n")

	fmt.Printf("Kind: ")
	kind, _ := reader.ReadString('\n')
	cfg.Kind = strings.TrimSuffix(kind, "\n")

	fmt.Printf("Platform9 Account URL: ")
	fqdn, _ := reader.ReadString('\n')
	cfg.Spec.AccountUrl = strings.TrimSuffix(fqdn, "\n")

	fmt.Printf("Username: ")
	username, _ := reader.ReadString('\n')
	cfg.Spec.Username = strings.TrimSuffix(username, "\n")

	fmt.Printf("Password: ")
	passwordBytes, _ := terminal.ReadPassword(0)
	password := base64.StdEncoding.EncodeToString(passwordBytes)
	cfg.Spec.Password = string(password)
	fmt.Println()

	fmt.Printf("Region [RegionOne]: ")
	region, _ := reader.ReadString('\n')
	cfg.Spec.Region = strings.TrimSuffix(region, "\n")

	fmt.Printf("Tenant [service]: ")
	service, _ := reader.ReadString('\n')
	cfg.Spec.Tenant = strings.TrimSuffix(service, "\n")

	fmt.Print("Proxy URL [None]: ")
	proxyURL, _ := reader.ReadString('\n')
	cfg.Spec.ProxyURL = strings.TrimSuffix(proxyURL, "\n")

	if cfg.Spec.Region == "" {
		cfg.Spec.Region = "RegionOne"
	}

	if cfg.Spec.Tenant == "" {
		cfg.Spec.Tenant = "service"
	}

	fmt.Print("MFA Token [None]: ")
	mfaToken, _ := reader.ReadString('\n')
	cfg.Spec.MfaToken = strings.TrimSuffix(mfaToken, "\n")

	var b []byte
	var err error
	if JsonFileType {
		b, err = json.MarshalIndent(cfg, "", " ")
	} else {
		b, err = yaml.Marshal(cfg)
	}

	if err != nil {
		fmt.Println(err)
		zap.S().Fatal("Error creating user config please try again")
	}

	if util.ConfigFileLoc != "" {
		if _, err := os.Stat(util.ConfigFileLoc); os.IsNotExist(err) {
			zap.S().Debugf("%s dir is not present creating it")
			err = os.MkdirAll(util.ConfigFileLoc, 0700)
			if err != nil {
				zap.S().Fatalf("error creating %s dir, please make sure dir is present", Location)
			}
		}
	} else {
		util.ConfigFileLoc = util.Pf9DBDir
	}

	if util.ConfigFileName != "" {
		FileName = filepath.Join(util.ConfigFileLoc, util.ConfigFileName)
	} else {
		FileName = filepath.Join(util.ConfigFileLoc, "config.json")
	}

	_, err = os.OpenFile(FileName, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error creating config file, please try again")
	}
	err = ioutil.WriteFile(FileName, b, 0644)
	if err == nil {
		fmt.Println("Config file created please check it here ", FileName)
	}

}

func CreateNodeConfig() {
	fmt.Println("Node config")
}

func CreateClusterConfig() {
	fmt.Println("Cluster config")
}
