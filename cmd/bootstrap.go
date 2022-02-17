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

// bootstrapCmd represents the bootstrap command
var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap [flags] cluster-name",
	Short: "Create a single node k8s cluster with current node",
	Long:  `Bootstrap a single node Kubernetes cluster with current node as the master node.`,
	Args: func(attachNodeCmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return errors.New("Only cluster name is accepted as a parameter")
		} else if len(args) < 1 {
			return errors.New("Cluster name is required for bootstrap")
		}
		clusterName = args[0]
		return nil
	},
	Run: bootstrapCmdRun,
}

var bootConfig objects.NodeConfig

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
	bootstrapCmd.MarkFlagRequired("pmk-version")
	bootstrapCmd.Flags().StringVar(&tag, "tag", "", "Add tag metadata to this cluster (key=value)")
	bootstrapCmd.Flags().StringVar(&interfaceDetection, "interface-detction-method", "first-found", "Interface detection method for Calico CNI")
	bootstrapCmd.Flags().StringVar(&ipEncapsulation, "ip-encapsulation", "Always", "Encapsulates POD traffic in IP-in-IP between nodes")
	bootstrapCmd.Flags().StringVar(&masterVIP, "master-virtual-ip", "", "Virtual IP address for cluster")
	bootstrapCmd.Flags().StringVar(&masterVIPIf, "master-virtual-interface", "", "Physical interface for virtual IP association")
	bootstrapCmd.Flags().BoolVar(&useHostName, "use-hostname", false, "Use node hostname for cluster creation")
	bootstrapCmd.Flags().BoolVar(&prometheusMonitoring, "monitoring", true, "Enable monitoring for this cluster")
	bootstrapCmd.Flags().BoolVar(&etcdBackup, "etcd-backup", true, "Enable automated etcd backups on this cluster")
	bootstrapCmd.Flags().BoolVar(&networkPluginOperator, "network-plugin-operator", false, "Will deploy Platform9 CRDs to enable multiple CNIs and features such as SR-IOV")
	bootstrapCmd.Flags().BoolVar(&enableKubVirt, "enable-kubeVirt", false, "Enables Kubernetes to run Virtual Machines within Pods. This feature is not recommended for production workloads")
	bootstrapCmd.Flags().BoolVar(&enableProfileEngine, "enable-profile-engine", true, "Simplfy cluster governance using the Platform9 Profile Engine")
	bootstrapCmd.Flags().StringVar(&metallbIPRange, "metallb-ip-range", "", "Ip range for MetalLB")
	bootstrapCmd.Flags().StringVar(&containersCIDR, "containers-cidr", "10.20.0.0/16", "CIDR for container overlay")
	bootstrapCmd.Flags().StringVar(&servicesCIDR, "services-cidr", "10.21.0.0/16", "CIDR for services overlay")
	bootstrapCmd.Flags().StringVar(&externalDNSName, "external-dns-name", "", "External DNS for master VIP")
	bootstrapCmd.Flags().BoolVar(&privileged, "privileged", true, "Enable privileged mode for K8s API. Default: true")
	bootstrapCmd.Flags().BoolVar(&allowWorkloadsOnMaster, "allow-workloads-on-master", true, "Taint master nodes ( to enable workloads )")
	bootstrapCmd.Flags().StringVar(&networkPlugin, "network-plugin", "calico", "Specify network plugin ( Possible values: flannel or calico )")
	bootstrapCmd.Flags().StringVarP(&bootConfig.User, "user", "u", "", "Ssh username for the node")
	bootstrapCmd.Flags().StringVarP(&bootConfig.Password, "password", "p", "", "Ssh password for the node (use 'single quotes' to pass password)")
	bootstrapCmd.Flags().StringVarP(&bootConfig.SshKey, "ssh-key", "s", "", "Ssh key file for connecting to the node")
	bootstrapCmd.Flags().StringSliceVarP(&bootConfig.IPs, "ip", "i", []string{}, "IP address of the host to be prepared")
	bootstrapCmd.Flags().StringVar(&bootConfig.MFA, "mfa", "", "MFA token")
	bootstrapCmd.Flags().StringVarP(&bootConfig.SudoPassword, "sudo-pass", "e", "", "Sudo password for user on remote host")
	rootCmd.AddCommand(bootstrapCmd)
}

var (
	useHostName              bool
	networkPluginOperator    bool
	enableKubVirt            bool
	prometheusMonitoring     bool
	etcdBackup               bool
	enableProfileEngine      bool
	networkStack             int
	apiServerFlags           []string
	controllerManagerFlags   []string
	schedulerFlags           []string
	tag                      string
	topologyManagerPolicy    string
	reservedCPUs             string
	containerRuntime         string
	mtuSize                  string
	blockSize                string
	pmkVersion               string
	ipEncapsulation          string
	interfaceDetection       string
	advancedAPIconfiguration string
	masterVIP                string
	masterVIPIf              string
	metallbIPRange           string
	containersCIDR           string
	servicesCIDR             string
	externalDNSName          string
	privileged               bool
	allowWorkloadsOnMaster   bool
	networkPlugin            string
	calicoNatOutgoing        int
)

func bootstrapCmdRun(cmd *cobra.Command, args []string) {
	zap.S().Debug("Received a call to bootstrap the node")

	detachedMode := cmd.Flags().Changed("no-prompt")
	isRemote := cmdexec.CheckRemote(bootConfig)

	isEtcdBackupDisabled := cmd.Flags().Changed("etcd-backup")
	qbert.IsMonitoringDisabled = cmd.Flags().Changed("monitoring")
	enabledKubVirt := cmd.Flags().Changed("enable-kubeVirt")
	if enabledKubVirt {
		networkPluginOperator = true
	}
	isIPv6enabled := cmd.Flags().Changed("network-stack")
	if isIPv6enabled {
		useHostName = false
		networkPlugin = "calico"
	}

	qbert.IsPMKversionDefined = cmd.Flags().Changed("pmk-version")
	if qbert.IsPMKversionDefined {
		//Profile Engine support check
		qbert.SplitPMKversion = strings.Split(pmkVersion, "-")
		if qbert.SplitPMKversion[0] < "1.20.11" {
			containerRuntime = "docker"
			enableProfileEngine = false
		}
	}

	qbert.IStag = cmd.Flags().Changed("tag")
	if qbert.IStag {
		qbert.SplitKeyValue = strings.Split(tag, "=")
	}

	if isRemote {
		if !config.ValidateNodeConfig(&bootConfig, !detachedMode) {
			zap.S().Fatal("Invalid remote node config (Username/Password/IP), use 'single quotes' to pass password")
		}
	}

	cfg := &objects.Config{WaitPeriod: time.Duration(60), AllowInsecure: false, MfaToken: bootConfig.MFA}
	var err error
	if detachedMode {
		err = config.LoadConfig(util.Pf9DBLoc, cfg, bootConfig)
	} else {
		err = config.LoadConfigInteractive(util.Pf9DBLoc, cfg, bootConfig)
	}
	if err != nil {
		zap.S().Fatalf("Unable to load the context: %s\n", err.Error())
	}

	fmt.Println(color.Green("✓ ") + "Loaded Config Successfully")

	var executor cmdexec.Executor
	if executor, err = cmdexec.GetExecutor(cfg.ProxyURL, bootConfig); err != nil {
		zap.S().Fatalf("Unable to create executor: %s\n", err.Error())
	}

	var c client.Client
	if c, err = client.NewClient(cfg.Fqdn, executor, cfg.AllowInsecure, false); err != nil {
		zap.S().Fatalf("Unable to create client: %s\n", err.Error())
	}

	defer c.Segment.Close()

	if isRemote {
		if err := SudoPasswordCheck(executor, detachedMode, bootConfig.SudoPassword); err != nil {
			zap.S().Fatal("Failed executing commands on remote machine with sudo: ", err.Error())
		}
	}

	defer c.Segment.Close()

	// Fetch the keystone token.
	auth, err := c.Keystone.GetAuth(
		cfg.Username,
		cfg.Password,
		cfg.Tenant,
		cfg.MfaToken,
	)

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Color("red")
	s.Start()
	defer s.Stop()
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

		result, err := pmk.CheckNode(*cfg, c, auth)
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
		LocalPath: "/etc/pf9/etcd-backup",
	}

	etcdDefaults := qbert.EtcdBackup{
		StorageType:         "local",
		IsEtcdBackupEnabled: 1,
		StorageProperties:   etcdBackupPath,
		IntervalInMins:      1440,
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

	if err := pmk.Bootstrap(*cfg, c, payload, auth, bootConfig); err != nil {

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
