// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"fmt"

	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/log"
	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/platform9/pf9ctl/pkg/qbert"
	"github.com/platform9/pf9ctl/pkg/supportBundle"
	"github.com/platform9/pf9ctl/pkg/util"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// bootstrapCmd represents the bootstrap command
var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Create a single node k8s cluster with your current node",
	Long: `Bootstrap a single node Kubernetes cluster with your current
	host as the Kubernetes node. Read more at
	http://pf9.io/cli_clbootstrap.`,

	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("Missing required argument: clusterName")
		}
		return nil
	},
	Run: bootstrapCmdRun,
}

var (
	masterVIP              string
	masterVIPIf            string
	metallbIPRange         string
	containersCIDR         string
	servicesCIDR           string
	externalDNSName        string
	privileged             bool
	appCatalogEnabled      bool
	allowWorkloadsOnMaster bool
	networkPlugin          string
)

func bootstrapCmdRun(cmd *cobra.Command, args []string) {
	zap.S().Debug("Received a call to bootstrap the node")

	zap.S().Debug("========== Running check-node as a part of bootup ==========")

	// This flag is used to loop back if user enters invalid credentials during config set.
	credentialFlag = true
	// To bail out if loop runs recursively more than thrice
	pmk.LoopCounter = 0

	for credentialFlag {

		ctx, err = pmk.LoadConfig(util.Pf9DBLoc)
		if err != nil {
			zap.S().Fatalf("Unable to load the context: %s\n", err.Error())
		}

		executor, err := getExecutor(ctx.ProxyURL)
		if err != nil {
			//debug first since Fatalf calls os.Exit
			zap.S().Debug("Error connecting to host %s", err.Error())
			zap.S().Fatalf(" Invalid (Username/Password/IP), use 'single quotes' to pass password")
		}

		c, err = pmk.NewClient(ctx.Fqdn, executor, ctx.AllowInsecure, false)
		if err != nil {
			zap.S().Fatalf("Unable to load clients needed for the Cmd. Error: %s", err.Error())
		}

		if FoundRemote {
			// Check if Remote Host needs Password to access Sudo
			SudoPasswordCheck(c.Executor)
		}

		// Validate the user credentials entered during config set and will loop back again if invalid
		if err := validateUserCredentials(ctx, c); err != nil {
			//Clearing the invalid config entered. So that it will ask for new information again.
			clearContext(&pmk.Context)
			//Check if no or invalid config exists, then bail out if asked for correct config for maxLoop times.
			err = configValidation(RegionInvalid, pmk.LoopCounter)
		} else {
			// We will store the set config if its set for first time using check-node
			if pmk.IsNewConfig {
				if err := pmk.StoreConfig(ctx, util.Pf9DBLoc); err != nil {
					zap.S().Errorf("Failed to store config: %s", err.Error())
				} else {
					pmk.IsNewConfig = false
				}
			}
			credentialFlag = false
		}
	}

	defer c.Segment.Close()

	result, err := pmk.CheckNode(ctx, c)
	if err != nil {
		// Uploads pf9cli log bundle if checknode fails
		errbundle := supportBundle.SupportBundleUpload(ctx, c)
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

	name := args[0]

	payload := qbert.ClusterCreateRequest{
		Name:                  name,
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

	err = pmk.Bootstrap(ctx, c, payload)
	if err != nil {
		c.Segment.SendEvent("Bootstrap : Cluster creation failed", err, "FAIL", "")
		zap.S().Fatalf("Unable to bootstrap the cluster. Error: %s", err.Error())
	}

	if err := c.Segment.SendEvent("Bootstrap : Cluster creation succeeded", payload, "PASS", ""); err != nil {
		zap.S().Errorf("Unable to send Segment event for Bootstrap. Error: %s", err.Error())
	}
}

func init() {
	bootstrapCmd.Flags().StringVar(&masterVIP, "masterVip", "", "IP Address for VIP for master nodes")
	bootstrapCmd.Flags().StringVar(&masterVIPIf, "masterVipIf", "", "Interface name for master / worker nodes")
	bootstrapCmd.Flags().StringVar(&metallbIPRange, "metallbIpRange", "", "Ip range for MetalLB")
	bootstrapCmd.Flags().StringVar(&containersCIDR, "containersCidr", "10.20.0.0/16", "CIDR for container overlay")
	bootstrapCmd.Flags().StringVar(&servicesCIDR, "servicesCidr", "10.21.0.0/16", "CIDR for services overlay")
	bootstrapCmd.Flags().StringVar(&externalDNSName, "externalDnsName", "", "External DNS for master VIP")
	bootstrapCmd.Flags().BoolVar(&privileged, "privileged", true, "Enable privileged mode for K8's API. Default: true")
	bootstrapCmd.Flags().BoolVar(&appCatalogEnabled, "appCatalogEnabled", false, "Enable Helm application catalog")
	bootstrapCmd.Flags().BoolVar(&allowWorkloadsOnMaster, "allowWorkloadsOnMaster", true, "Taint master nodes ( to enable workloads )")
	bootstrapCmd.Flags().StringVar(&networkPlugin, "networkPlugin", "flannel", "Specify network plugin ( Possible values: flannel or calico )")
	// This is the bootstrap command to initialize its run and add to root which is not in use for now.
	rootCmd.AddCommand(bootstrapCmd)
}
