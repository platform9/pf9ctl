package cmd

import (
	"fmt"

	"github.com/platform9/pf9ctl/pkg/client"
	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/config"
	"github.com/platform9/pf9ctl/pkg/util"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var listClusterCmd = &cobra.Command{
	Use:   "list",
	Short: "List all the clusters associated with your PMK account",
	Long:  "List all the clusters associated with your PMK account",
	Run:   listClusterCmdRun,
}

func init() {
	clusterCmd.AddCommand(listClusterCmd)
}

func listClusterCmdRun(cmd *cobra.Command, args []string) {
	zap.S().Debug("==========Running cluster list==========")
	detachedMode := cmd.Flags().Changed("no-prompt")

	var err error
	if detachedMode {
		err = config.LoadConfig(util.Pf9DBLoc, cfg)
	} else {
		err = config.LoadConfigInteractive(util.Pf9DBLoc, cfg)
	}
	if err != nil {
		zap.S().Fatalf("Unable to load the context: %s\n", err.Error())
	}
	fmt.Println(color.Green("✓ ") + "Loaded Config Successfully")

	var executor cmdexec.Executor
	if executor, err = cmdexec.GetExecutor(cfg.Spec.ProxyURL, util.Node, nc); err != nil {
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
	clusters := c.Qbert.ListCluster(auth.Token, auth.ProjectID)
	if len(clusters) < 1 {
		zap.S().Info("No clusters are available")
	} else {
		fmt.Println("Available clusters are")
		for _, v := range clusters {
			fmt.Println(v.Name)
		}
	}
}
