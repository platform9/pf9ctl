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
	Short: "Creates a single-node Kubernetes cluster using the current node",
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
	bootstrapCmd.Flags().StringVar(&masterVIP, "master-vip", "", "IP Address for VIP for master nodes")
	bootstrapCmd.Flags().StringVar(&masterVIPIf, "master-vip-if", "", "Interface name for master / worker nodes")
	bootstrapCmd.Flags().StringVar(&metallbIPRange, "metallb-ip-range", "", "Ip range for MetalLB")
	bootstrapCmd.Flags().StringVar(&containersCIDR, "containers-cidr", "10.20.0.0/16", "CIDR for container overlay")
	bootstrapCmd.Flags().StringVar(&servicesCIDR, "services-cidr", "10.21.0.0/16", "CIDR for services overlay")
	bootstrapCmd.Flags().StringVar(&externalDNSName, "external-dns-name", "", "External DNS for master VIP")
	bootstrapCmd.Flags().BoolVar(&privileged, "privileged", true, "Enable privileged mode for K8s API. Default: true")
	bootstrapCmd.Flags().BoolVar(&allowWorkloadsOnMaster, "allow-workloads-on-master", true, "Taint master nodes ( to enable workloads )")
	bootstrapCmd.Flags().StringVar(&networkPlugin, "network-plugin", "calico", "Specify network plugin ( Possible values: flannel or calico )")
	bootstrapCmd.Flags().StringVarP(&bootConfig.User, "user", "u", "", "ssh username for the nodes")
	bootstrapCmd.Flags().StringVarP(&bootConfig.Password, "password", "p", "", "ssh password for the nodes (use 'single quotes' to pass password)")
	bootstrapCmd.Flags().StringVarP(&bootConfig.SshKey, "ssh-key", "s", "", "ssh key file for connecting to the nodes")
	bootstrapCmd.Flags().StringSliceVarP(&bootConfig.IPs, "ip", "i", []string{}, "IP address of host to be prepared")
	bootstrapCmd.Flags().StringVar(&bootConfig.MFA, "mfa", "", "MFA token")
	bootstrapCmd.Flags().StringVarP(&bootConfig.SudoPassword, "sudo-pass", "e", "", "sudo password for user on remote host")
	bootstrapCmd.Flags().BoolVarP(&bootConfig.RemoveExistingPkgs, "remove-existing-pkgs", "r", false, "Will remove previous installation if found (default false)")
	rootCmd.AddCommand(bootstrapCmd)
}

var (
	masterVIP              string
	masterVIPIf            string
	metallbIPRange         string
	containersCIDR         string
	servicesCIDR           string
	externalDNSName        string
	privileged             bool
	allowWorkloadsOnMaster bool
	networkPlugin          string
)

func bootstrapCmdRun(cmd *cobra.Command, args []string) {
	zap.S().Debug("Received a call to bootstrap the node")

	detachedMode := cmd.Flags().Changed("no-prompt")
	isRemote := cmdexec.CheckRemote(bootConfig)

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

		result, err := pmk.CheckNode(*cfg, c, auth, bootConfig)
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

	payload := qbert.ClusterCreateRequest{
		Name:                  clusterName,
		ContainerCIDR:         containersCIDR,
		ServiceCIDR:           servicesCIDR,
		MasterVirtualIP:       masterVIP,
		MasterVirtualIPIface:  masterVIPIf,
		ExternalDNSName:       externalDNSName,
		NetworkPlugin:         qbert.CNIBackend(networkPlugin),
		MetalLBAddressPool:    metallbIPRange,
		AllowWorkloadOnMaster: allowWorkloadsOnMaster,
		Privileged:            privileged,
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
