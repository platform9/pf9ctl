package cmd

import (
	"errors"
	"fmt"

	"github.com/platform9/pf9ctl/pkg/client"
	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/config"
	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/platform9/pf9ctl/pkg/util"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	clusterUuid string
)

var deleteClusterCmd = &cobra.Command{
	Use:   "delete-cluster",
	Short: "Deletes the cluster",
	Long:  "Deletes the cluster with the specified name. Additionally the user can pass the cluster UID instead of the name.",
	Args: func(deauthNodeCmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errors.New("No parameters are needed")
		}
		return nil
	},
	Run: deleteClusterRun,
}

func init() {
	deleteClusterCmd.Flags().StringVarP(&clusterName, "name", "n", "", "clusters name")
	deleteClusterCmd.Flags().StringVarP(&clusterUuid, "uuid", "i", "", "clusters uuid")
	deleteClusterCmd.Flags().StringVar(&util.MFA, "mfa", "", "MFA token")
	clusterCmd.AddCommand(deleteClusterCmd)
}

func deleteClusterRun(cmd *cobra.Command, args []string) {

	if !cmd.Flags().Changed("name") && !cmd.Flags().Changed("uuid") {
		zap.S().Fatalf("You must pass a cluster name or the cluster uuid")
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

	auth, err := c.Keystone.GetAuth(cfg.Spec.Username, cfg.Spec.Password, cfg.Spec.Tenant, cfg.Spec.MfaToken)
	if err != nil {
		zap.S().Debug("Failed to get keystone %s", err.Error())
	}

	projectId := auth.ProjectID
	token := auth.Token

	if !cmd.Flags().Changed("uuid") {
		_, clusterUuid, _, err = c.Qbert.CheckClusterExists(clusterName, projectId, token)

		if err != nil {
			zap.S().Fatalf("Could not delete the cluster")
		}

	}

	nodeIPs = append(nodeIPs, pmk.GetIp().String())

	projectNodes := c.Qbert.GetAllNodes(token, projectId)
	nodeUuids := c.Resmgr.GetHostId(token, nodeIPs)
	localNode, err := getNodesFromUuids(nodeUuids, projectNodes)

	if len(localNode) == 1 && localNode[0].ClusterUuid == clusterUuid {
		c.Executor.RunCommandWait("sudo pkill -9 `pidof kubelet`")
		c.Executor.RunCommandWait("sudo pkill -9 `pidof etcd`")
		c.Executor.RunCommandWait("sudo pkill -9 `pidof kube-proxy`")
	}

	err = c.Qbert.DeleteCluster(clusterUuid, projectId, token)
	if err != nil {
		zap.S().Fatalf("Error deleting cluster ", err.Error())
	}
	fmt.Println("Cluster deletion started....This may take a few minutes.")
	zap.S().Debug("Cluster deletion started....This may take a few minutes.")
}
