// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"fmt"
	"log"

	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/platform9/pf9ctl/pkg/util"
	"github.com/spf13/cobra"
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
	RunE: bootstrapCmdRun,
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

func bootstrapCmdRun(cmd *cobra.Command, args []string) error {
	log.Println("Received a call to bootstrap the node")

	ctx, err := pmk.LoadContext(pmk.Pf9DBLoc)
	if err != nil {
		return err
	}

	name := args[0]
	_, _ = pmk.NewClusterCreate(
		name,
		containersCIDR,
		servicesCIDR,
		masterVIP,
		masterVIPIf,
		externalDNSName,
		networkPlugin,
		metallbIPRange,
		allowWorkloadsOnMaster,
		privileged,
	)

	resp, err := util.AskBool("PrepLocal node for kubernetes cluster")
	if err != nil || !resp {
		return fmt.Errorf("Couldn't fetch user content")
	}

	err = pmk.PrepNode(ctx, "", "", "", []string{})
	if err != nil {
		return fmt.Errorf("Unable to prepnode: %s", err.Error())
	}

	keystoneAuth, err := pmk.GetKeystoneAuth(ctx.Fqdn, ctx.Username, ctx.Password, ctx.Tenant)
	uuid, err := pmk.GetNodePoolUUID(ctx, keystoneAuth)
	if err != nil {
		return err
	}
	fmt.Println(uuid)

	// **TODO**: cluster attach

	return nil
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

	rootCmd.AddCommand(bootstrapCmd)
}
