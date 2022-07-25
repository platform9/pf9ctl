package cmd

import (
	"errors"
	"fmt"

	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/config"
	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/platform9/pf9ctl/pkg/util"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var decommissionNodeCmd = &cobra.Command{
	Use:   "decommission",
	Short: "Decommissions this node from the PMK control plane",
	Long:  "Removes the host agent package and decommissions this node from the Platform9 control plane.",
	Args: func(deauthNodeCmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errors.New("No parameters are needed")
		}
		return nil
	},
	Run: decommissionNodeRun,
	PreRun: func(cmd *cobra.Command, args []string) {
		if util.Node.Hostname != "" {
			nc.Spec.Nodes = append(nc.Spec.Nodes, util.Node)
		}
	},
}

func init() {
	decommissionNodeCmd.Flags().StringVar(&util.MFA, "mfa", "", "MFA token")
	decommissionNodeCmd.Flags().StringVarP(&util.Node.Hostname, "user", "u", "", "ssh username for the nodes")
	decommissionNodeCmd.Flags().StringVarP(&nc.Password, "password", "p", "", "ssh password for the nodes (use 'single quotes' to pass password)")
	decommissionNodeCmd.Flags().StringVarP(&nc.SshKey, "ssh-key", "s", "", "ssh key file for connecting to the nodes")
	decommissionNodeCmd.Flags().StringVarP(&util.Node.Ip, "ip", "i", "", "IP address of host to be decommissioned")
	decommissionNodeCmd.Flags().StringVar(&ConfigPath, "user-config", "", "Path of user-config file")
	decommissionNodeCmd.Flags().StringVar(&NodeConfigPath, "node-config", "", "Path of node-config file")
	nodeCmd.AddCommand(decommissionNodeCmd)
}

func decommissionNodeRun(cmd *cobra.Command, args []string) {

	if cmd.Flags().Changed("user-config") {
		util.Pf9DBLoc = ConfigPath
	}

	if cmd.Flags().Changed("node-config") {
		config.LoadNodeConfig(nc, NodeConfigPath)
		util.Node = nc.Spec.Nodes[0]
	}

	detachedMode := cmd.Flags().Changed("no-prompt")

	if cmdexec.CheckRemote(util.Node) {
		/*if !config.ValidateNodeConfig(host, nc, !detachedMode) {
			zap.S().Fatal("Invalid remote node config (Username/Password/IP), use 'single quotes' to pass password")
		}*/
	}

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
	zap.S().Debug("Loaded Config Successfully")
	pmk.DecommissionNode(cfg, nc, true)

}
