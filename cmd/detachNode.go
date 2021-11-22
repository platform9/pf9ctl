package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/platform9/pf9ctl/pkg/client"
	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/config"
	"github.com/platform9/pf9ctl/pkg/objects"
	"github.com/platform9/pf9ctl/pkg/util"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type Node struct {
	Uuid        string `json:"uuid"`
	ClusterUuid string `json:"clusterUuid"`
	PrimaryIp   string `json:"primaryIp"`
	IsMaster    string `json:"isMaster"`
}

var (
	nodeIPs []string
)

var detachNodeCmd = &cobra.Command{
	Use:   "detach-node [flags]",
	Short: "detaches a node from a kubernetes cluster",
	Long:  "Detach nodes from their clusters. If no nodes are passed it will detach the node on which the command was run.",
	Args: func(detachNodeCmd *cobra.Command, args []string) error {
		return nil
	},
	Run: detachNodeRun,
}

func init() {
	detachNodeCmd.Flags().StringSliceVarP(&nodeIPs, "node-ip", "n", []string{}, "node ip address")
	detachNodeCmd.Flags().StringVar(&attachconfig.MFA, "mfa", "", "MFA token")
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

	if len(nodeIPs) == 0 {
		nodeIPs = append(nodeIPs, getIp().String())
	}

	detachedMode := cmd.Flags().Changed("no-prompt")

	if cmdexec.CheckRemote(nc) {
		if !config.ValidateNodeConfig(&nc, !detachedMode) {
			zap.S().Fatal("Invalid remote node config (Username/Password/IP), use 'single quotes' to pass password")
		}
	}

	cfg := &objects.Config{WaitPeriod: time.Duration(60), AllowInsecure: false, MfaToken: attachconfig.MFA}
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

	var executor cmdexec.Executor
	if executor, err = cmdexec.GetExecutor(cfg.ProxyURL, nc); err != nil {
		zap.S().Fatalf("Unable to create executor: %s\n", err.Error())
	}

	var c client.Client
	if c, err = client.NewClient(cfg.Fqdn, executor, cfg.AllowInsecure, false); err != nil {
		zap.S().Fatalf("Unable to create client: %s\n", err.Error())
	}

	defer c.Segment.Close()

	auth, err := c.Keystone.GetAuth(cfg.Username, cfg.Password, cfg.Tenant, cfg.MfaToken)
	if err != nil {
		zap.S().Debug("Failed to get keystone %s", err.Error())
	}
	projectId := auth.ProjectID
	token := auth.Token

	projectNodes := getAllProjectNodes(c.Executor, cfg.Fqdn, token, projectId)

	nodeUuids, _ := hostId(c.Executor, cfg.Fqdn, token, nodeIPs)

	detachNodes := getNodesFromUuids(nodeUuids, projectNodes)

	fmt.Println("Starting detaching process")
	if err := c.Segment.SendEvent("Starting detach-node", auth, "", ""); err != nil {
		zap.S().Errorf("Unable to send Segment event for detach node. Error: %s", err.Error())
	}

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
			zap.S().Infof("Node [%v] detached from cluster", detachNodes[i].Uuid)
		}
	}

}

func getAllProjectNodes(exec cmdexec.Executor, fqdn string, token string, projectID string) []Node {
	zap.S().Debug("Getting cluster status")
	tkn := fmt.Sprintf(`"X-Auth-Token: %v"`, token)
	cmd := fmt.Sprintf("curl -sH %v -X GET %v/qbert/v3/%v/nodes", tkn, fqdn, projectID)
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
