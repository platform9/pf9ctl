package cmd

import (
	"fmt"

	"github.com/platform9/pf9ctl/pkg/client"
	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/config"
	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/platform9/pf9ctl/pkg/qbert"
	"github.com/platform9/pf9ctl/pkg/util"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	nodeIPs []string
)

var detachNodeCmd = &cobra.Command{
	Use:   "detach [flags]",
	Short: "Detaches a node from a Kubernetes cluster",
	Long:  "Detach nodes from their clusters. If no nodes are passed it will detach the node on which the command was run.",
	Args: func(detachNodeCmd *cobra.Command, args []string) error {
		return nil
	},
	Run: detachNodeRun,
}

func init() {
	detachNodeCmd.Flags().StringSliceVarP(&nodeIPs, "node-ip", "n", []string{}, "node ip address")
	detachNodeCmd.Flags().StringVar(&util.MFA, "mfa", "", "MFA token")
	nodeCmd.AddCommand(detachNodeCmd)
}

func detachNodeRun(cmd *cobra.Command, args []string) {

	if len(nodeIPs) == 0 {
		nodeIPs = append(nodeIPs, pmk.GetIp().String())
	}

	detachedMode := cmd.Flags().Changed("no-prompt")

	if cmdexec.CheckRemote(nc) {
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

	auth, err := c.Keystone.GetAuth(cfg.Spec.Username, cfg.Spec.Password, cfg.Spec.Tenant, cfg.Spec.MfaToken)
	if err != nil {
		zap.S().Debug("Failed to get keystone %s", err.Error())
	}
	projectId := auth.ProjectID
	token := auth.Token

	projectNodes := c.Qbert.GetAllNodes(token, projectId)
	nodeUuids := c.Resmgr.GetHostId(token, nodeIPs)
	if err != nil {
		zap.S().Fatalf("%v", err)
		return
	}

	detachNodes, err := getNodesFromUuids(nodeUuids, projectNodes)

	if err != nil {
		zap.S().Fatalf(err.Error())
	}

	fmt.Println("Starting detaching process")

	if err := c.Segment.SendEvent("Starting detach-node", auth, "", ""); err != nil {
		zap.S().Debugf("Unable to send Segment event for detach node. Error: %s", err.Error())
	}

	for i := range detachNodes {

		isMaster := c.Qbert.GetNodeInfo(token, projectId, nodeUuids[0])
		clusterNodes := getAllClusterNodes(projectNodes, []string{isMaster.ClusterUuid})

		if len(clusterNodes) == 1 || isMaster.IsMaster == 1 {
			fmt.Printf("Node %v is either the master node or the last node in the cluster\n", isMaster.Uuid)
		}

		err1 := c.Qbert.DetachNode(detachNodes[i].ClusterUuid, projectId, token, detachNodes[i].Uuid)

		if err1 != nil {
			if err := c.Segment.SendEvent("Detaching-node", auth, "Failed to detach node", ""); err != nil {
				zap.S().Debugf("Unable to send Segment event for detach node. Error: %s", err.Error())
			}
			zap.S().Info("Encountered an error while detaching the", detachNodes[i].PrimaryIp, " node from a Kubernetes cluster : ", err1)
		} else {
			if err := c.Segment.SendEvent("Detaching-node", detachNodes[i].PrimaryIp, "Node detached", ""); err != nil {
				zap.S().Debugf("Unable to send Segment event for detach node. Error: %s", err.Error())
			}
			zap.S().Infof("Node [%v] detached from cluster", detachNodes[i].Uuid)
		}
	}

}

//returns the nodes whos ip's were passed in the flag (or the node installed on the machine if no ip was passed)
func getNodesFromUuids(nodeUuids []string, allNodes []qbert.Node) ([]qbert.Node, error) {

	var nodesUuid []qbert.Node
	for i := range allNodes {
		for j := range nodeUuids {

			if nodeUuids[j] == allNodes[i].Uuid {

				if allNodes[i].ClusterUuid == "" {
					return nodesUuid, fmt.Errorf("The node %v is not connected to any clusters", allNodes[i].PrimaryIp)
				} else {
					nodesUuid = append(nodesUuid, allNodes[i])
					break
				}
			}
		}
	}
	return nodesUuid, nil
}

//returns a list of all clusters the nodes are attached to
func getClusters(allNodes []qbert.Node) []string {

	var clusters []string

	for i := range allNodes {
		clusterExists := false
		for j := range clusters {
			if clusters[j] == allNodes[i].ClusterUuid {
				clusterExists = true
				break
			}
		}
		if !clusterExists {
			clusters = append(clusters, allNodes[i].ClusterUuid)
		}

	}

	return clusters

}

//returns all nodes attached to a specific clusters, used to detach all nodes from clusters
func getAllClusterNodes(allNodes []qbert.Node, clusters []string) []qbert.Node {

	var clusterNodes []qbert.Node

	for i := range allNodes {

		for j := range clusters {
			if allNodes[i].ClusterUuid == clusters[j] {
				clusterNodes = append(clusterNodes, allNodes[i])
				break
			}
		}

	}
	return clusterNodes
}
