// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/platform9/pf9ctl/pkg/log"
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
	log.Info.Println("Received a call to bootstrap the node")

	ctx, err := pmk.LoadContext(pmk.Pf9DBLoc)
	if err != nil {
		return err
	}
	name := args[0]

	resp, err := util.AskBool("PrepLocal node for kubernetes cluster")
	if err != nil || !resp {
		return fmt.Errorf("Couldn't fetch user content")
	}

	err = pmk.PrepNode(ctx, "", "", "", []string{})
	if err != nil {
		return fmt.Errorf("Unable to prepnode: %s", err.Error())
	}

	cluster, _ := pmk.NewCluster(
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
	keystoneAuth, err := pmk.GetKeystoneAuth(
		ctx.Fqdn,
		ctx.Username,
		ctx.Password,
		ctx.Tenant)

	if err != nil {
		return fmt.Errorf("keystone authentication failed: %s", err.Error())
	}

	_, err = cluster.Create(ctx, keystoneAuth)
	if err != nil {
		return fmt.Errorf("Unable to create cluster: %s", err.Error())
	}

	c := `cat /etc/pf9/host_id.conf | grep ^host_id | cut -d = -f2 | cut -d ' ' -f2`
	nodeUUID, err := exec.Command("bash", "-c", c).Output()
	nodeUUIDStr := strings.TrimSuffix(string(nodeUUID), "\n")

	log.Info.Println("Waiting for the cluster to get created")
	time.Sleep(pmk.WaitPeriod * time.Second)

	log.Info.Println("Cluster created successfully")
	err = cluster.AttachNode(ctx, keystoneAuth, nodeUUIDStr)
	if err != nil {
		return fmt.Errorf("Unable to attach node: %s", err.Error())
	}

	log.Info.Printf("\nBootstrap successfully Finished\n")
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
