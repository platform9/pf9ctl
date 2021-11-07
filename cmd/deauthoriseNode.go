package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
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

var deauthNodeCmd = &cobra.Command{
	Use:   "deauthorize-node",
	Short: "Deauthorizes this node from the Platform9 control plane",
	Long:  "Deauthorizes a node. You can aurhotize it again by using the authorize-node command.",
	Args: func(deauthNodeCmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errors.New("No parameters are needed")
		}
		return nil
	},
	Run: deauthNodeRun,
}

func init() {
	rootCmd.AddCommand(deauthNodeCmd)
}

func deauthNodeRun(cmd *cobra.Command, args []string) {

	detachedMode := cmd.Flags().Changed("dt")

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

	auth, err := c.Keystone.GetAuth(cfg.Username, cfg.Password, cfg.Tenant, cfg.MfaToken)
	if err != nil {
		zap.S().Debug("Failed to get keystone %s", err.Error())
	}

	var nodeIPs []string
	nodeIPs = append(nodeIPs, getIp().String())
	projectId := auth.ProjectID
	token := auth.Token
	nodeUuids, _ := hostId(c.Executor, cfg.Fqdn, token, nodeIPs)

	if len(nodeUuids) == 0 {
		zap.S().Fatalf("Could not find the node. Check if the node associated with this account")
	}

	isMaster := getNode(c.Executor, cfg.Fqdn, token, projectId, nodeUuids[0])

	projectNodes := getAllProjectNodes(c.Executor, cfg.Fqdn, token, projectId)

	clusterNodes := getAllClusterNodes(projectNodes, []string{isMaster.ClusterUuid})

	removeCluster := false

	if len(clusterNodes) == 1 || isMaster.IsMaster == "1" {
		removeCluster = true
	}

	err = c.Qbert.DeauthoriseNode(isMaster.Uuid, token)

	if err != nil {
		zap.S().Fatalf("Error deauthorising node ", err.Error())
	}

	fmt.Println("Node deauthorized")

	if removeCluster {
		err = c.Qbert.DeleteCluster(isMaster.ClusterUuid, projectId, token)
		if err != nil {
			zap.S().Fatalf("Error deleting cluster ", err.Error())
		}
		fmt.Println("The cluster was deleted")
	}

}

func getNode(exec cmdexec.Executor, fqdn string, token string, projectID string, nodeUuid string) Node {
	zap.S().Debug("Checking if node is master")
	tkn := fmt.Sprintf(`"X-Auth-Token: %v"`, token)
	cmd := fmt.Sprintf(`curl -sH %v -X GET %v/qbert/v3/%v/nodes | jq -r '.[] | select(.uuid=="`+nodeUuid+`")' `, tkn, fqdn, projectID)
	isMaster, err := exec.RunWithStdout("bash", "-c", cmd)
	if err != nil {
		zap.S().Fatalf("Unable to get node status: ", err)
	}

	var nodes Node
	json.Unmarshal([]byte(isMaster), &nodes)

	return nodes
}
