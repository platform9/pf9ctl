// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"fmt"

	"github.com/platform9/pf9ctl/pkg/constants"
	"github.com/platform9/pf9ctl/pkg/log"
	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/platform9/pf9ctl/pkg/pmk/clients"
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
	log.Debug("Received a call to bootstrap the node")

	ctx, err := pmk.LoadContext(constants.Pf9DBLoc)
	if err != nil {
		log.Fatalf("Unable to load context: %s", err.Error())
	}

	c, err := clients.New(ctx.Fqdn, clients.LocalExecutor{})
	if err != nil {
		log.Fatalf("Unable to load clients: %s", err.Error())
	}
	defer c.Segment.Close()

	name := args[0]

	payload := clients.ClusterCreateRequest{
		Name:                  name,
		ContainerCIDR:         containersCIDR,
		ServiceCIDR:           servicesCIDR,
		MasterVirtualIP:       masterVIP,
		MasterVirtualIPIface:  masterVIPIf,
		ExternalDNSName:       externalDNSName,
		NetworkPlugin:         clients.CNIBackend(networkPlugin),
		MetalLBAddressPool:    metallbIPRange,
		AllowWorkloadOnMaster: allowWorkloadsOnMaster,
		Privileged:            privileged,
	}

	err = pmk.Bootstrap(ctx, c, payload)
	if err != nil {
		c.Segment.SendEvent("Bootstrap - Cluster creation failed", err)
		log.Fatalf("Unable to bootstrap the cluster. Error: %s", err.Error())
	}

	if err := c.Segment.SendEvent("Bootstrap - Cluster creation succeeded", payload); err != nil {
		log.Errorf("Unable to send Segment event for Bootstrap. Error: %s", err.Error())
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

	rootCmd.AddCommand(bootstrapCmd)
}
