package cmd

import (
	"errors"
	"fmt"
	"time"

	"github.com/platform9/pf9ctl/pkg/client"
	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/config"
	"github.com/platform9/pf9ctl/pkg/objects"
	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/platform9/pf9ctl/pkg/util"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var deauthNodeCmd = &cobra.Command{
	Use:   "deauthorize-node",
	Short: "Deauthorizes this node from the PMK control plane",
	Long:  "Deauthorizes this node. It will warn the user if the node was a master node or a part of a single node cluster.",
	Args: func(deauthNodeCmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errors.New("No parameters are needed")
		}
		return nil
	},
	Run: deauthNodeRun,
}

func init() {
	deauthNodeCmd.Flags().StringVar(&attachconfig.MFA, "mfa", "", "MFA token")
	deauthNodeCmd.Flags().StringVarP(&ipAdd, "ip", "i", "", "IP address of the host to be deauthorized")
	rootCmd.AddCommand(deauthNodeCmd)
}

func deauthNodeRun(cmd *cobra.Command, args []string) {

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
	zap.S().Debug("Loaded Config Successfully")
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
	if ipAdd != "" {
		nodeIPs = append(nodeIPs, ipAdd)
	} else {
		nodeIPs = append(nodeIPs, pmk.GetIp().String())
	}
	projectId := auth.ProjectID
	token := auth.Token
	nodeUuids := c.Resmgr.GetHostId(token, nodeIPs)
	if len(nodeUuids) == 0 {
		zap.S().Fatalf("Could not find the node. Check if the node associated with this account")
	}

	isMaster := c.Qbert.GetNodeInfo(token, projectId, nodeUuids[0])

	if !detachedMode && isMaster.ClusterUuid != "" {

		projectNodes := c.Qbert.GetAllNodes(token, projectId)
		clusterNodes := getAllClusterNodes(projectNodes, []string{isMaster.ClusterUuid})

		if len(clusterNodes) == 1 || isMaster.IsMaster == 1 {
			fmt.Println("Warning: The node is either the master node or the last node in the cluster.")
			fmt.Print("Do you still want to deauthorize it?")
			answer, err := util.AskBool("")
			if err != nil {
				zap.S().Fatalf("Stopping deauthorization")
			}
			if !answer {
				fmt.Println("Stopping deauthorization")
				return
			}

		}

	}

	err = c.Qbert.DeauthoriseNode(isMaster.Uuid, token)

	if err != nil {
		node := c.Qbert.GetNodeInfo(token, projectId, nodeUuids[0])
		if node.Uuid == "" {
			zap.S().Infof("Node might be already deauthorized, please check in UI")
		}
		zap.S().Fatalf("Error deauthorising node ", err.Error())
	}

	fmt.Println("Node deauthorization started....This may take a few minutes....Check the latest status in UI")
	zap.S().Debug("Node deauthorization started....This may take a few minutes....Check the latest status in UI")
}
