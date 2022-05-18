// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/platform9/pf9ctl/pkg/client"
	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/config"
	"github.com/platform9/pf9ctl/pkg/log"
	"github.com/platform9/pf9ctl/pkg/objects"
	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/platform9/pf9ctl/pkg/qbert"
	"github.com/platform9/pf9ctl/pkg/supportBundle"
	"github.com/platform9/pf9ctl/pkg/util"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const boostrapHelpTemplate = `
Bootstrap a single node Kubernetes cluster with current node as the master node.

Usage:
  pf9ctl bootstrap [flags] cluster-name

Examples:
pf9ctl bootstrap <clusterName> --pmk-version <version>

Required Flags:
	    --pmk-version string                  Kubernetes pmk version
Optional Flags:
	    --advanced-api-configuration string   Allowed API groups and version. Option: default, all & custom
	    --allow-workloads-on-master           Taint master nodes ( to enable workloads ), use either --allow-workloads-on-master or --allow-workloads-on-master=false to change (default true)
	    --api-server-flags strings            Comma separated list of supported kube-apiserver flags, e.g: --request-timeout=2m0s,--kubelet-timeout=20s
	    --block-size string                   Block size determines how many Pod's can run per node vs total number of nodes per cluster (default "26")
	    --container-runtime string            The container runtime for the cluster (default "containerd")
	    --containers-cidr string              CIDR for container overlay (default "10.20.0.0/16")
	    --controller-manager-flags strings    Comma separated list of supported kube-controller-manager flags, e.g: --large-cluster-size-threshold=60,--concurrent-statefulset-syncs=10
	    --enable-kubeVirt                     Enables Kubernetes to run Virtual Machines within Pods. This feature is not recommended for production workloads, use either --enable-kubeVirt or --enable-kubeVirt=true to change
	    --enable-profile-engine               Simplfy cluster governance using the Platform9 Profile Engine, use either --enable-profile-engine or --enable-profile-engine=false to change (default true)
	    --etcd-backup                         Enable automated etcd backups on this cluster, use either --etcd-backup or --etcd-backup=false to change (default true)
	    --etcd-backup-path string             Backup path for etcd (default "/etc/pf9/etcd-backup")
	    --external-dns-name string            External DNS for master VIP
	-h, --help                                help for bootstrap
	    --http-proxy string                   Specify the HTTP proxy for this cluster. Format-> <scheme>://<username>:<password>@<host>:<port>, username and password are optional.
	    --interface-detction-method string    Interface detection method for Calico CNI (default "first-found")
	    --interval-in-mins                    Time interval of etcd-backup in minutes(should be between 30 to 60) (default 30)
	-i, --ip strings                          IP address of the host to be prepared
	    --ip-encapsulation string             Encapsulates POD traffic in IP-in-IP between nodes (default "Always")
	    --master-virtual-interface string     Physical interface for virtual IP association
	    --master-virtual-ip string            Virtual IP address for cluster
	    --metallb-ip-range string             Ip range for MetalLB
	    --mfa string                          MFA token
	    --monitoring                          Enable monitoring for this cluster, use either --monitoring or --monitoring=false to change (default true)
	    --mtu-size string                     Maximum Transmission Unit (MTU) for the interface (default "1440")
	    --nat int                             Packets destined outside the POD network will be SNAT'd using the node's IP (default 1)
	    --network-plugin string               Specify network plugin ( Possible values: flannel or calico ) (default "calico")
	    --network-plugin-operator             Will deploy Platform9 CRDs to enable multiple CNIs and features such as SR-IOV, use either --network-plugin-operator or --network-plugin-operator=true to change
	    --network-stack int                   0 for ipv4 and 1 for ipv6
	-p, --password string                     Ssh password for the node (use 'single quotes' to pass password)
	    --privileged                          Enable privileged mode for K8s API, use either --privileged or --privileged=false to change (default true)
	-r, --remove-existing-pkgs                Will remove previous installation if found, use either --remove-existing-pkgs or --remove-existing-pkgs=true to change
	    --reserved-cpu string                 Comma separated list of CPUs to be reserved for the system, e.g: 4-8,9-12
	    --scheduler-flags strings             Comma separated list of supported Kube-scheduler flags, e.g: --kube-api-burst=120,--log_file_max_size=3000
	    --services-cidr string                CIDR for services overlay (default "10.21.0.0/16")
	-s, --ssh-key string                      Ssh key file for connecting to the node
	-e, --sudo-pass string                    Sudo password for user on remote host
	    --tag string                          Add tag metadata to this cluster (key=value)
            --topology-manager-policy string      Topology manager policy (default "none")
	    --use-hostname                        Use node hostname for cluster creation, use either --use-hostname or --use-hostname=true to change
	-u, --user string                         Ssh username for the node


Global Flags:
	    --log-dir string   path to save logs
	    --no-prompt        disable all user prompts
	    --verbose          print verbose logs

`

// bootstrapCmd represents the bootstrap command
var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap [flags] cluster-name",
	Short: "Creates a single-node Kubernetes cluster using the current node",
	Long:  `Bootstrap a single node Kubernetes cluster with current node as the master node.`,
	Args: func(bootstrapCmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return errors.New("Only cluster name is accepted as a parameter")
		} else if len(args) < 1 {
			return errors.New("Cluster name is required for bootstrap")
		}
		clusterName = args[0]
		return nil
	},

	PreRun: func(cmd *cobra.Command, args []string) {
		if node.Hostname != "" {
			nc.Spec.Nodes = append(nc.Spec.Nodes, node)
		}
	},

	Example: "pf9ctl bootstrap <clusterName> --pmk-version <version>",
	Run:     bootstrapCmdRun,
}

var node objects.Node

func init() {
	bootstrapCmd.Flags().IntVar(&networkStack, "network-stack", 0, "0 for ipv4 and 1 for ipv6")
	bootstrapCmd.Flags().StringVar(&containerRuntime, "container-runtime", "containerd", "The container runtime for the cluster")
	bootstrapCmd.Flags().IntVar(&calicoNatOutgoing, "nat", 1, "Packets destined outside the POD network will be SNAT'd using the node's IP")
	bootstrapCmd.Flags().StringVar(&mtuSize, "mtu-size", "1440", "Maximum Transmission Unit (MTU) for the interface")
	bootstrapCmd.Flags().StringVar(&blockSize, "block-size", "26", "Block size determines how many Pod's can run per node vs total number of nodes per cluster")
	bootstrapCmd.Flags().StringVar(&topologyManagerPolicy, "topology-manager-policy", "none", "Topology manager policy")
	bootstrapCmd.Flags().StringVar(&reservedCPUs, "reserved-cpu", "", "Comma separated list of CPUs to be reserved for the system, e.g: 4-8,9-12")
	bootstrapCmd.Flags().StringSliceVarP(&apiServerFlags, "api-server-flags", "", []string{}, "Comma separated list of supported kube-apiserver flags, e.g: --request-timeout=2m0s,--kubelet-timeout=20s")
	bootstrapCmd.Flags().StringSliceVarP(&controllerManagerFlags, "controller-manager-flags", "", []string{}, "Comma separated list of supported kube-controller-manager flags, e.g: --large-cluster-size-threshold=60,--concurrent-statefulset-syncs=10")
	bootstrapCmd.Flags().StringSliceVarP(&schedulerFlags, "scheduler-flags", "", []string{}, "Comma separated list of supported Kube-scheduler flags, e.g: --kube-api-burst=120,--log_file_max_size=3000")
	bootstrapCmd.Flags().StringVar(&advancedAPIconfiguration, "advanced-api-configuration", "", "Allowed API groups and version. Option: default, all & custom")
	bootstrapCmd.Flags().StringVar(&pmkVersion, "pmk-version", "", "Kubernetes pmk version")
	bootstrapCmd.Flags().StringVar(&tag, "tag", "", "Add tag metadata to this cluster (key=value)")
	bootstrapCmd.Flags().StringVar(&interfaceDetection, "interface-detction-method", "first-found", "Interface detection method for Calico CNI")
	bootstrapCmd.Flags().StringVar(&ipEncapsulation, "ip-encapsulation", "Always", "Encapsulates POD traffic in IP-in-IP between nodes")
	bootstrapCmd.Flags().StringVar(&masterVIP, "master-virtual-ip", "", "Virtual IP address for cluster")
	bootstrapCmd.Flags().StringVar(&masterVIPIf, "master-virtual-interface", "", "Physical interface for virtual IP association")
	bootstrapCmd.Flags().BoolVar(&useHostName, "use-hostname", false, "Use node hostname for cluster creation, use either --use-hostname or --use-hostname=true to change")
	bootstrapCmd.Flags().BoolVar(&prometheusMonitoring, "monitoring", true, "Enable monitoring for this cluster, use either --monitoring or --monitoring=false to change")
	bootstrapCmd.Flags().BoolVar(&etcdBackup, "etcd-backup", true, "Enable automated etcd backups on this cluster, use either --etcd-backup or --etcd-backup=false to change")
	bootstrapCmd.Flags().BoolVar(&networkPluginOperator, "network-plugin-operator", false, "Will deploy Platform9 CRDs to enable multiple CNIs and features such as SR-IOV, use either --network-plugin-operator or --network-plugin-operator=true to change")
	bootstrapCmd.Flags().BoolVar(&enableKubVirt, "enable-kubeVirt", false, "Enables Kubernetes to run Virtual Machines within Pods. This feature is not recommended for production workloads, use either --enable-kubeVirt or --enable-kubeVirt=true to change")
	bootstrapCmd.Flags().BoolVar(&enableProfileEngine, "enable-profile-engine", true, "Simplfy cluster governance using the Platform9 Profile Engine, use either --enable-profile-engine or --enable-profile-engine=false to change")
	bootstrapCmd.Flags().StringVar(&metallbIPRange, "metallb-ip-range", "", "Ip range for MetalLB")
	bootstrapCmd.Flags().StringVar(&containersCIDR, "containers-cidr", "10.20.0.0/16", "CIDR for container overlay")
	bootstrapCmd.Flags().StringVar(&servicesCIDR, "services-cidr", "10.21.0.0/16", "CIDR for services overlay")
	bootstrapCmd.Flags().StringVar(&externalDNSName, "external-dns-name", "", "External DNS for master VIP")
	bootstrapCmd.Flags().BoolVar(&privileged, "privileged", true, "Enable privileged mode for K8s API, use either --privileged or --privileged=false to change")
	bootstrapCmd.Flags().BoolVar(&allowWorkloadsOnMaster, "allow-workloads-on-master", true, "Taint master nodes ( to enable workloads ), use either --allow-workloads-on-master or --allow-workloads-on-master=false to change")
	bootstrapCmd.Flags().StringVar(&networkPlugin, "network-plugin", "calico", "Specify network plugin ( Possible values: flannel or calico )")
	bootstrapCmd.Flags().StringVarP(&node.Hostname, "user", "u", "", "Ssh username for the node")
	bootstrapCmd.Flags().StringVarP(&nc.Password, "password", "p", "", "Ssh password for the node (use 'single quotes' to pass password)")
	bootstrapCmd.Flags().StringVarP(&nc.SshKey, "ssh-key", "s", "", "Ssh key file for connecting to the node")
	bootstrapCmd.Flags().StringVarP(&node.Ip, "ip", "i", "", "IP address of the host to be prepared")
	bootstrapCmd.Flags().StringVar(&util.MFA, "mfa", "", "MFA token")
	bootstrapCmd.Flags().StringVarP(&util.SudoPassword, "sudo-pass", "e", "", "Sudo password for user on remote host")
	bootstrapCmd.Flags().BoolVarP(&util.RemoveExistingPkgs, "remove-existing-pkgs", "r", false, "Will remove previous installation if found (default false)")
	bootstrapCmd.Flags().StringVar(&httpProxy, "http-proxy", "", "Specify the HTTP proxy for this cluster. Format-> <scheme>://<username>:<password>@<host>:<port>, username and password are optional.")
	bootstrapCmd.Flags().StringVar(&ConfigPath, "user-config", "", "Path of user-config file")
	bootstrapCmd.Flags().StringVar(&NodeConfigPath, "node-config", "", "Path of node-config file")
	bootstrapCmd.Flags().IntVar(&intervalInMins, "interval-in-mins", 30, "time interval of etcd-backup in minutes(should be between 30 to 60)")
	bootstrapCmd.Flags().StringVar(&backupPath, "etcd-backup-path", "/etc/pf9/etcd-backup", "Backup path for etcd")
	bootstrapCmd.SetHelpTemplate(boostrapHelpTemplate)

	clusterCmd.AddCommand(bootstrapCmd)
}

var (
	useHostName              bool     //if set then hostname will be used for cluster creation
	networkPluginOperator    bool     //if set then network plugin operator (add-on) will be enabled on cluster
	enableKubVirt            bool     //if set then kubeVirt (add-on) will be enabled on cluster
	prometheusMonitoring     bool     //if set then monitoring (add-on) will be enabled on cluster
	etcdBackup               bool     //if set then etcd back-up will be enabled with default values
	enableProfileEngine      bool     //if set then profile engine will be enabled on cluster
	networkStack             int      //if set then IPv6 network stack will be used
	apiServerFlags           []string //takes list of api server flags for cluster creation
	controllerManagerFlags   []string //takes list of controller manager flags for cluster creation
	schedulerFlags           []string //takes list of scheduler flags for cluster creation
	tag                      string   //add tag metadata to this cluster creaton
	topologyManagerPolicy    string   //to topology manager support
	reservedCPUs             string   //CPUs to be reserved for the system
	containerRuntime         string   //the container runtime for the cluster
	mtuSize                  string   //determines how many Pod's can run per node vs total number of nodes per cluster
	blockSize                string   //maximum transmission unit (MTU) for the interface (in bytes)
	pmkVersion               string   //pmk role version
	ipEncapsulation          string   //ip encapsulation mode
	interfaceDetection       string   //interface detection method
	advancedAPIconfiguration string   //Kubernetes API configuration
	masterVIP                string   //Virtual IP address for cluster
	masterVIPIf              string   //Physical interface for virtual IP association
	metallbIPRange           string   //Ip range for MetalLB
	containersCIDR           string   //containersCIDR ip
	servicesCIDR             string   //servicesCIDR ip
	externalDNSName          string   //External DNS for master VIP
	privileged               bool     //if set then this allows this cluster to run privileged containers
	allowWorkloadsOnMaster   bool     //if set then workloads on master nodes is allowed
	networkPlugin            string   //cluster CNI network backend
	calicoNatOutgoing        int      //if set then packets destined outside the POD network will be SNAT'd using the node's IP.
	httpProxy                string   //the HTTP proxy for this cluster.
	intervalInMins           int      //etcd backup interval in minutes
	backupPath               string   //etcd storage path
)

func bootstrapCmdRun(cmd *cobra.Command, args []string) {
	zap.S().Debug("Received a call to bootstrap the node")

	if cmd.Flags().Changed("user-config") {
		util.Pf9DBLoc = ConfigPath
	}

	if cmd.Flags().Changed("node-config") {
		config.LoadNodeConfig(nc, NodeConfigPath)
	}

	detachedMode := cmd.Flags().Changed("no-prompt")
	isRemote := cmdexec.CheckRemote(nc)

	isEtcdBackupDisabled := cmd.Flags().Changed("etcd-backup")
	qbert.IsMonitoringDisabled = cmd.Flags().Changed("monitoring")
	//if set then network plugin operator is enabled
	enabledKubVirt := cmd.Flags().Changed("enable-kubeVirt")
	if enabledKubVirt {
		networkPluginOperator = true
	}
	isIPv6enabled := cmd.Flags().Changed("network-stack")
	//IPv6 only supports calico, and by default node ip will be used for cluster creaton
	if isIPv6enabled {
		useHostName = false
		networkPlugin = util.Calico
	}

	qbert.IStag = cmd.Flags().Changed("tag")
	if qbert.IStag {
		qbert.SplitKeyValue = strings.Split(tag, "=")
	}

	if isRemote {
		if !config.ValidateNodeConfig(nc, !detachedMode) {
			zap.S().Fatal("Invalid remote node config (Username/Password/IP), use 'single quotes' to pass password")
		}
	}

	var err error
	if detachedMode {
		err = config.LoadConfig(util.Pf9DBLoc, cfg, nc)
	} else {
		err = config.LoadConfigInteractive(util.Pf9DBLoc, cfg, nc)
	}
	if err != nil {
		zap.S().Fatalf("Unable to load the context: %s\n", err.Error())
	}

	fmt.Println(color.Green("✓ ") + "Loaded Config Successfully")
	zap.S().Debug("Loaded Config Successfully")
	var executor cmdexec.Executor
	if executor, err = cmdexec.GetExecutor(cfg.Spec.ProxyURL, nc); err != nil {
		zap.S().Fatalf("Unable to create executor: %s\n", err.Error())
	}

	var c client.Client
	if c, err = client.NewClient(cfg.Spec.AccountUrl, executor, cfg.Spec.OtherData.AllowInsecure, false); err != nil {
		zap.S().Fatalf("Unable to create client: %s\n", err.Error())
	}

	defer c.Segment.Close()

	if isRemote {
		if err := SudoPasswordCheck(executor, detachedMode, util.SudoPassword); err != nil {
			zap.S().Fatal("Failed executing commands on remote machine with sudo: ", err.Error())
		}
	}

	defer c.Segment.Close()

	// Fetch the keystone token.
	auth, err := c.Keystone.GetAuth(
		cfg.Spec.Username,
		cfg.Spec.Password,
		cfg.Spec.Tenant,
		cfg.Spec.MfaToken,
	)

	//Getting all pmk versions
	pmkRoles := c.Qbert.GetPMKVersions(auth.Token, auth.ProjectID)

	qbert.IsPMKversionDefined = cmd.Flags().Changed("pmk-version")
	if qbert.IsPMKversionDefined {
		//Profile Engine support check, Profile engine is supported for 1.20.11 and above versions
		qbert.SplitPMKversion = strings.Split(pmkVersion, "-")
		if qbert.SplitPMKversion[0] < util.PmkVersion {
			enableProfileEngine = false
		}
		//Selected Docker as default container runtime for pmk version 1.20.11 and below versions
		if qbert.SplitPMKversion[0] <= util.PmkVersion {
			containerRuntime = util.Docker
		}
	} else {
		fmt.Printf("supported pmk versions are\n")
		for _, v := range pmkRoles.Roles {
			fmt.Println(v.RoleVersion)
		}
		zap.S().Fatalf("pmk-version is mandatory, please specify pmk version")
	}

	var versionNotFound bool
	for _, v := range pmkRoles.Roles {
		if v.RoleVersion != pmkVersion {
			versionNotFound = true
		} else {
			versionNotFound = false
			break
		}
	}

	if versionNotFound {
		fmt.Printf("supported pmk versions are\n")
		for _, v := range pmkRoles.Roles {
			fmt.Println(v.RoleVersion)
		}
		zap.S().Fatalf("%s pmk-version is not supported", pmkVersion)
	}

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Color("red")
	s.Start()
	defer s.Stop()
	zap.S().Debug("Running pre-requisite checks for Bootstrap command")
	s.Suffix = " Running pre-requisite checks for Bootstrap command"

	val, val1, err := pmk.PreReqBootstrap(executor)
	if err != nil {
		zap.S().Fatalf("Error running Prerequisite Checks for Bootstrap Command")
	}
	s.Stop()
	if !val1 && !val { //Both node and cluster are already present
		zap.S().Fatalf(color.Red("x ") + " Cannot run this command as this node is already attached to a cluster")

	} else if !val && val1 { //Only node is present but not attached to a cluster
		util.SkipPrepNode = true
		fmt.Println(color.Green("✓") + " Node is already Onboarded....Skipping Prep-Node")
	} else { //Both node and cluster are not present
		fmt.Println(color.Green("✓") + " Node is not onboarded and not attached to any cluster")
		util.SkipPrepNode = false
	}

	if !util.SkipPrepNode {
		zap.S().Debug("========== Running check-node as a part of bootstrap ==========")

		result, err := pmk.CheckNode(*cfg, c, auth, nc)
		if err != nil {
			// Uploads pf9cli log bundle if checknode fails
			errbundle := supportBundle.SupportBundleUpload(*cfg, c, isRemote)
			if errbundle != nil {
				zap.S().Debugf("Unable to upload supportbundle to s3 bucket %s", errbundle.Error())
			}
			zap.S().Fatalf("Unable to perform pre-requisite checks on this node: %s", err.Error())
		}

		if result == pmk.RequiredFail {
			zap.S().Fatalf(color.Red("x ")+"Required pre-requisite check(s) failed. See %s or use --verbose for logs \n", log.GetLogLocation(util.Pf9Log))
			//this is so the exit flag is set to 1
		} else if result == pmk.OptionalFail {
			fmt.Printf("\nOptional pre-requisite check(s) failed. See %s or use --verbose for logs \n", log.GetLogLocation(util.Pf9Log))
		} else if result == pmk.CleanInstallFail {
			fmt.Println("\nPrevious Installation Removed")
		}

		zap.S().Debug("==========Finished running check-node==========")

		zap.S().Debug("Received a call to boostrap the local node")

		if !detachedMode {
			resp, err := util.AskBool("Prep local node as master node for kubernetes cluster")
			if err != nil || !resp {
				zap.S().Fatalf(" Declined to proceed with creating a Kubernetes cluster with the current node as the master node ")
			}
		} else {
			fmt.Println(" Proceeding to create a Kubernetes cluster with current node as master node")
		}

		zap.S().Debug("========== Running prep-node as a part of bootstrap ==========")
		if err := pmk.PrepNode(*cfg, c, auth); err != nil {

			// Uploads pf9cli log bundle if prepnode failed to get prepared
			errbundle := supportBundle.SupportBundleUpload(*cfg, c, isRemote)
			if errbundle != nil {
				zap.S().Debugf("Unable to upload supportbundle to s3 bucket %s", errbundle.Error())
			}

			zap.S().Debugf("Unable to prep node: %s\n", err.Error())
			zap.S().Fatalf("\nFailed to prepare node. See %s or use --verbose for logs\n", log.GetLogLocation(util.Pf9Log))
		}

		zap.S().Debug("==========Finished running prep-node==========")
	}
	defer c.Segment.Close()

	etcdBackupPath := qbert.Storageproperties{
		LocalPath: backupPath,
	}

	etcdDefaults := qbert.EtcdBackup{
		StorageType:            "local",
		IsEtcdBackupEnabled:    1,
		StorageProperties:      etcdBackupPath,
		IntervalInMins:         intervalInMins,
		MaxIntervalBackupCount: 3,
	}
	if isEtcdBackupDisabled {
		etcdDefaults = qbert.EtcdBackup{}
	}

	payload := qbert.ClusterCreateRequest{
		Name:                   clusterName,
		ContainerCIDR:          containersCIDR,
		ServiceCIDR:            servicesCIDR,
		MasterVirtualIP:        masterVIP,
		MasterVirtualIPIface:   masterVIPIf,
		ExternalDNSName:        externalDNSName,
		NetworkPlugin:          qbert.CNIBackend(networkPlugin),
		MetalLBAddressPool:     metallbIPRange,
		AllowWorkloadOnMaster:  allowWorkloadsOnMaster,
		Privileged:             privileged,
		EtcdBackup:             etcdDefaults,
		NetworkPluginOperator:  networkPluginOperator,
		EnableKubVirt:          enableKubVirt,
		EnableProfileAgent:     enableProfileEngine,
		PmkVersion:             pmkVersion,
		IPEncapsulation:        ipEncapsulation,
		InterfaceDetection:     interfaceDetection,
		UseHostName:            useHostName,
		MtuSize:                mtuSize,
		BlockSize:              blockSize,
		ContainerRuntime:       containerRuntime,
		NetworkStack:           networkStack,
		TopologyManagerPolicy:  topologyManagerPolicy,
		ReservedCPUs:           reservedCPUs,
		ApiServerFlags:         apiServerFlags,
		ControllerManagerFlags: controllerManagerFlags,
		SchedulerFlags:         schedulerFlags,
		RuntimeConfig:          advancedAPIconfiguration,
		CalicoNatOutgoing:      calicoNatOutgoing,
		HttpProxy:              httpProxy,
	}

	if err != nil {
		// Certificate expiration is detected by the http library and
		// only error object gets populated, which means that the http
		// status code does not reflect the actual error code.
		// So parsing the err to check for certificate expiration.
		if strings.Contains(strings.ToLower(err.Error()), util.CertsExpireErr) {

			zap.S().Fatalf("Possible clock skew detected. Check the system time and retry.")
		}
		zap.S().Fatalf("Unable to obtain keystone credentials: %s", err.Error())
	}

	if err := pmk.Bootstrap(*cfg, c, payload, auth, nc); err != nil {

		// Uploads pf9cli log bundle if bootstrap command fails
		errbundle := supportBundle.SupportBundleUpload(*cfg, c, isRemote)
		if errbundle != nil {
			zap.S().Debugf("Unable to upload supportbundle to s3 bucket %s", errbundle.Error())
		}

		zap.S().Debugf("Unable to bootstrap node: %s\n", err.Error())
		zap.S().Fatalf("Failed to bootstrap node. See %s or use --verbose for logs\n", log.GetLogLocation(util.Pf9Log))
	}
	zap.S().Debug("==========Finished running bootstrap==========")
}
