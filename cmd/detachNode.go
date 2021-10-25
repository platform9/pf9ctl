package cmd

import (
	"errors"
	"fmt"
	"log"
	"net"

	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/platform9/pf9ctl/pkg/util"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	nodeIPs []string
)

var detachNodeCmd = &cobra.Command{
	Use:   "detach-node [flags] cluster-name",
	Short: "detaches a node from a kubernetes cluster",
	Long:  "Detach a node from an existing cluster.",
	Args: func(detachNodeCmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return errors.New("Only cluster name is accepted as a parameter")
		} else if len(args) < 1 {
			return errors.New("Cluster name is required for attach-node")
		}
		clusterName = args[0]
		return nil
	},
	Run: detachNodeRun,
}

func init() {
	detachNodeCmd.Flags().StringSliceVarP(&nodeIPs, "node-ip", "n", []string{}, "node ip address")
	rootCmd.AddCommand(detachNodeCmd)
}

func getIp() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

func detachNodeRun(cmd *cobra.Command, args []string) {
	zap.S().Debug("==========Running Attach Node==========")
	// This flag is used to loop back if user enters invalid credentials during config set.
	credentialFlag = true
	// To bail out if loop runs recursively more than thrice
	pmk.LoopCounter = 0

	if len(nodeIPs) == 0 {
		nodeIPs = append(nodeIPs, getIp().String())
	}

	for credentialFlag {

		ctx, err = pmk.LoadConfig(util.Pf9DBLoc)
		if err != nil {
			zap.S().Fatalf("Unable to load the context: %s\n", err.Error())
		}

		executor, err := getExecutor(ctx.ProxyURL)
		if err != nil {
			zap.S().Debug("Error connecting to host %s", err.Error())
			zap.S().Fatalf(" Invalid (Username/Password/IP)")
		}

		c, err = pmk.NewClient(ctx.Fqdn, executor, ctx.AllowInsecure, false)
		if err != nil {
			zap.S().Fatalf("Unable to load clients needed for the Cmd. Error: %s", err.Error())
		}
		// Validate the user credentials entered during config set and will loop back again if invalid
		if err := validateUserCredentials(ctx, c); err != nil {
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

	auth, err := c.Keystone.GetAuth(ctx.Username, ctx.Password, ctx.Tenant, ctx.MfaToken)
	if err != nil {
		zap.S().Debug("Failed to get keystone %s", err.Error())
	}
	projectId := auth.ProjectID
	token := auth.Token

	node_hostIds, err := hostId(c.Executor, ctx.Fqdn, token, nodeIPs)

	_, cluster_uuid, _ := c.Qbert.CheckClusterExists(clusterName, projectId, token)
	clusterStatus := cluster_Status(c.Executor, ctx.Fqdn, token, projectId, cluster_uuid)
	if clusterStatus == "ok" {
		fmt.Println("Starting detaching process")
		if err := c.Segment.SendEvent("Starting Dettach-node", auth, "", ""); err != nil {
			zap.S().Errorf("Unable to send Segment event for detach node. Error: %s", err.Error())
		}
		err1 := c.Qbert.DetachNode(cluster_uuid, projectId, token, node_hostIds)
		if err1 != nil {
			if err := c.Segment.SendEvent("Detaching-node", auth, "Failed to detach node", ""); err != nil {
				zap.S().Errorf("Unable to send Segment event for detach node. Error: %s", err.Error())
			}
			zap.S().Info("Encountered an error while detaching node from a Kubernetes cluster : ", err1)
		} else {
			if err := c.Segment.SendEvent("Detaching-node", auth, "Node detached", ""); err != nil {
				zap.S().Errorf("Unable to send Segment event for detach node. Error: %s", err.Error())
			}
			zap.S().Infof("Node(s) %v detached  from cluster", nodeIPs)
		}

	} else {
		zap.S().Fatalf("Cluster is not ready. cluster status is %v", clusterStatus)
	}

}
