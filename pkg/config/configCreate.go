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

	/*fmt.Printf("API version: ")
	apiVersion, _ := reader.ReadString('\n')
	cfg.ApiVersion = strings.TrimSuffix(apiVersion, "\n")

	fmt.Printf("Kind: ")
	kind, _ := reader.ReadString('\n')
	cfg.Kind = strings.TrimSuffix(kind, "\n")*/

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

	ifDirNotExistCreat()
	createConfigFile("config.json", b)
}

func CreateNodeConfig() {
	cfg := objects.NodeConfig{}

	reader := bufio.NewReader(os.Stdin)
	/*fmt.Printf("API version: ")
	apiVersion, _ := reader.ReadString('\n')
	cfg.APIVersion = strings.TrimSuffix(apiVersion, "\n")

	fmt.Printf("Kind: ")
	kind, _ := reader.ReadString('\n')
	cfg.Kind = strings.TrimSuffix(kind, "\n")

	fmt.Printf("DeploymentKind: ")
	Dkind, _ := reader.ReadString('\n')
	cfg.Spec.DeploymentKind = strings.TrimSuffix(Dkind, "\n")

	fmt.Printf("Type: ")
	typ, _ := reader.ReadString('\n')
	cfg.Spec.Type = strings.TrimSuffix(typ, "\n")*/

	fmt.Printf("IP: ")
	ip, _ := reader.ReadString('\n')
	ip = strings.TrimSuffix(ip, "\n")

	fmt.Printf("HostName: ")
	hostName, _ := reader.ReadString('\n')
	hostName = strings.TrimSuffix(hostName, "\n")

	/*fmt.Printf("Type: ")
	Ntyp, _ := reader.ReadString('\n')
	Ntyp = strings.TrimSuffix(Ntyp, "\n")*/

	node := objects.Node{
		Ip:       ip,
		Hostname: hostName,
	}

	cfg.Spec.Nodes = append(cfg.Spec.Nodes, node)

	fmt.Printf("SSH-Key: ")
	key, _ := reader.ReadString('\n')
	cfg.SshKey = strings.TrimSuffix(key, "\n")

	var b []byte
	var err error
	if JsonFileType {
		b, err = json.MarshalIndent(cfg, "", " ")
	} else {
		b, err = yaml.Marshal(cfg)
	}

	if err != nil {
		fmt.Println(err)
		zap.S().Fatal("Error creating node config please try again")
	}

	ifDirNotExistCreat()
	createConfigFile("NodeConfig", b)
}

func CreateClusterConfig() {
	clusterConfig := objects.ClusterConfig{
		APIVersion: "v4",
		Kind:       "cluster",
		Spec: objects.ClusterSpec{
			ClusterSetting: objects.ClusterInfo{
				Name:            "pf9-cluster",
				KubeRoleVersion: "1.21.3-pmk.111",
			},
			ApplicationContainerSetting: objects.ContainerSetting{
				Previleged:            true,
				AllowWorkloadOnMaster: true,
				ContainerRuntime:      "Docker",
			},
			NetworkingAndRegistration: objects.Networking{
				ClusterNetworkStack: objects.NetworkStack{
					IPv4: true,
				},
				NodeRegistration: objects.Registration{
					UseNodeIPForClusterCreation: true,
				},
			},
			ClusterAddOns: objects.ClusterAddon{
				EtcdBackup: objects.EtcdBackup{
					StorageProperties: objects.StorageProperties{
						LocalPath: "/etc/pf9/etcd-backup",
					},
					DailyBackupTime:         "02:00",
					MaxTimestampBackupCount: 3,
				},
				Monitoring: objects.Monitoring{
					RetentionTime: "7d",
				},
				EnableProfileAgent: true,
			},
			ClusterNetworkInfo: objects.ClusterNetworkInfo{
				ContainerCIDR: "10.20.0.0/16",
				ServiceCIDR:   "10.21.0.0/16",
				ClusterCNI: objects.ClusterCNI{
					NetworkBackend:     "Calico",
					IPEncapsulation:    "Always",
					InterfaceDetection: "First Round",
				},
				NatOutgoing: objects.NatOutgoing{
					BlockSize: "26",
					MtuSize:   "1440",
				},
			},
		},
	}

	var b []byte
	var err error
	if JsonFileType {
		b, err = json.MarshalIndent(clusterConfig, "", " ")
	} else {
		b, err = yaml.Marshal(clusterConfig)
	}

	if err != nil {
		fmt.Println(err)
		zap.S().Fatal("Error creating cluster config please try again")
	}

	ifDirNotExistCreat()
	createConfigFile("ClusterConfig", b)

}

func ifDirNotExistCreat() {
	//If user have given dir location to store congig file, this function will check if that location is present,
	//if not then it will create it,
	//If user has not given any location it will store config in default db (pf9/db) file
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
}

func createConfigFile(name string, b []byte) {
	//this function will create config file
	if util.ConfigFileName != "" {
		FileName = filepath.Join(util.ConfigFileLoc, util.ConfigFileName)
	} else {
		FileName = filepath.Join(util.ConfigFileLoc, name)
	}

	_, err := os.OpenFile(FileName, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error creating config file, please try again")
	}
	err = ioutil.WriteFile(FileName, b, 0644)
	if err == nil {
		fmt.Println("Config file created please check it here ", FileName)
	}
}
