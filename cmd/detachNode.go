package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"net"

	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/platform9/pf9ctl/pkg/util"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type Node struct {
	Uuid        string `json:"uuid"`
	ClusterUuid string `json:"clusterUuid"`
	PrimaryIp   string `json:"primaryIp"`
}

var (
	nodeIPs       []string
	deleteCluster bool
)

var detachNodeCmd = &cobra.Command{
	Use:   "detach-node [flags]",
	Short: "detaches a node from a kubernetes cluster",
	Long:  "Detach a node from an existing cluster. Pass aditional parametar to also delete the cluster.",
	Args: func(detachNodeCmd *cobra.Command, args []string) error {
		return nil
	},
	Run: detachNodeRun,
}

func init() {
	detachNodeCmd.Flags().StringSliceVarP(&nodeIPs, "node-ip", "n", []string{}, "node ip address")
	detachNodeCmd.Flags().BoolVarP(&deleteCluster, "delete-cluster", "d", false, "if true will also delete nodes cluster")
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

	//node_hostIds, err := hostId(c.Executor, ctx.Fqdn, token, nodeIPs)

	projectNodes := getAllProjectNodes(c.Executor, ctx.Fqdn, token, projectId)
	//fmt.Println("PRoject nodes ", projectNodes)

	nodeUuids, _ := hostId(c.Executor, ctx.Fqdn, token, nodeIPs)

	detachNodes := getNodesFromUuids(nodeUuids, projectNodes)
	//fmt.Println("PRoject nodes ", nodesFromIP)

	clusters := getClusters(detachNodes)
	//fmt.Println("Clusters ", clusters)

	if deleteCluster {
		detachNodes = getAllClusterNodes(projectNodes, clusters)
	}

	fmt.Println("Starting detaching process")
	if err := c.Segment.SendEvent("Starting detach-node", auth, "", ""); err != nil {
		zap.S().Errorf("Unable to send Segment event for detach node. Error: %s", err.Error())
	}

	if deleteCluster {
		for i := range clusters {
			err1 := c.Qbert.DeleteCluster(clusters[i], projectId, token)

			if err1 != nil {
				if err := c.Segment.SendEvent("Deleting cluster", auth, "Failed to delete cluster", ""); err != nil {
					zap.S().Errorf("Unable to send Segment event for delete cluster. Error: %s", err.Error())
				}
				zap.S().Info("Encountered an error while deleting the ", clusters[i], " cluster: ", err1)
			} else {
				if err := c.Segment.SendEvent("Deleting cluster", clusters[i], "Cluster deleted", ""); err != nil {
					zap.S().Errorf("Unable to send Segment event for deleting cluster. Error: %s", err.Error())
				}
				zap.S().Infof("Cluster %v deleted", clusters[i])
			}
		}
	} else {

		for i := range detachNodes {
			err1 := c.Qbert.DetachNode(detachNodes[i].ClusterUuid, projectId, token, detachNodes[i].Uuid)

			if err1 != nil {
				if err := c.Segment.SendEvent("Detaching-node", auth, "Failed to detach node", ""); err != nil {
					zap.S().Errorf("Unable to send Segment event for detach node. Error: %s", err.Error())
				}
				zap.S().Info("Encountered an error while detaching the", detachNodes[i].PrimaryIp, " node from a Kubernetes cluster : ", err1)
			} else {
				if err := c.Segment.SendEvent("Detaching-node", detachNodes[i].PrimaryIp, "Node detached", ""); err != nil {
					zap.S().Errorf("Unable to send Segment event for detach node. Error: %s", err.Error())
				}
				zap.S().Infof("Node %v detached from cluster", detachNodes[i].PrimaryIp)
			}
		}
	}

}

func getAllProjectNodes(exec cmdexec.Executor, fqdn string, token string, projectID string) []Node {
	zap.S().Debug("Getting cluster status")
	tkn := fmt.Sprintf(`"X-Auth-Token: %v"`, token)
	cmd := fmt.Sprintf("curl -sH %v -X GET %v/qbert/v4/%v/nodes", tkn, fqdn, projectID)
	status, err := exec.RunWithStdout("bash", "-c", cmd)
	if err != nil {
		zap.S().Fatalf("Unable to get project nodes: ", err)
	}
	var nodes []Node
	json.Unmarshal([]byte(status), &nodes)

	return nodes
}

//returns the nodes whos ip's were passed in the flag (or the node installed on the machine if no ip was passed)
func getNodesFromUuids(nodeUuids []string, allNodes []Node) []Node {

	var nodesUuid []Node
	for i := range allNodes {
		for j := range nodeUuids {
			if nodeUuids[j] == allNodes[i].Uuid && allNodes[i].ClusterUuid != "" {
				nodesUuid = append(nodesUuid, allNodes[i])
				break
			}
		}
	}
	return nodesUuid
}

//returns a list of all clusters the nodes are attached to
func getClusters(allNodes []Node) []string {

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
func getAllClusterNodes(allNodes []Node, clusters []string) []Node {

	var clusterNodes []Node

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

//transforms an array of the Node object into an array of its Uuid attribute
func getNodesUuids(nodes []Node) []string {

	var nodeUuids []string
	for i := range nodes {
		nodeUuids = append(nodeUuids, nodes[i].Uuid)
	}
	return nodeUuids

}
