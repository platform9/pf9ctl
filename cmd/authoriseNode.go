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

var authNodeCmd = &cobra.Command{
	Use:   "authorize-node",
	Short: "Authorizes this node with PMK control plane",
	Long:  "Authorizes this node.",
	Args: func(deauthNodeCmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errors.New("No parameters are needed")
		}
		return nil
	},
	Run: authNodeRun,
}
var ipAdd string

func init() {
	rootCmd.AddCommand(authNodeCmd)
	authNodeCmd.Flags().StringVarP(&ipAdd, "ip", "i", "", "IP address of the host to be authorized")
	authNodeCmd.Flags().StringVar(&util.MFA, "mfa", "", "MFA token")
}

func authNodeRun(cmd *cobra.Command, args []string) {

	detachedMode := cmd.Flags().Changed("no-prompt")

	if cmdexec.CheckRemote(nc) {
		if !config.ValidateNodeConfig(&nc, !detachedMode) {
			zap.S().Fatal("Invalid remote node config (Username/Password/IP), use 'single quotes' to pass password")
		}
	}

	//cfg := &objects.UserData{OtherData: objects.Other{WaitPeriod: time.Duration(60), AllowInsecure: false}, MfaToken: attachconfig.MFA}
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

	var nodeIPs []string
	if ipAdd != "" {
		nodeIPs = append(nodeIPs, ipAdd)
	} else {
		nodeIPs = append(nodeIPs, pmk.GetIp().String())
	}
	token := auth.Token
	nodeUuids := c.Resmgr.GetHostId(token, nodeIPs)

	if len(nodeUuids) == 0 {
		zap.S().Fatalf("Could not find the node. Check if the node associated with this account")
	}

	err1 := c.Qbert.AuthoriseNode(nodeUuids[0], token)

	if err1 != nil {
		zap.S().Fatalf("Error authorising node ", err1.Error())
	}

	fmt.Println("Node authorization started....This may take a few minutes....Check the latest status in UI")

}
