package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/config"
	"github.com/platform9/pf9ctl/pkg/objects"
	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/platform9/pf9ctl/pkg/util"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var decommissionNodeCmd = &cobra.Command{
	Use:   "decommission-node",
	Short: "Decomisisons this node from the Platform9 control plane",
	Long:  "Removes the host agent package and decommissions this node from the Platform9 control plane.",
	Args: func(deauthNodeCmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errors.New("No parameters are needed")
		}
		return nil
	},
	Run: decommissionNodeRun,
}

func init() {
	decommissionNodeCmd.Flags().StringVar(&attachconfig.MFA, "mfa", "", "MFA token")
	rootCmd.AddCommand(decommissionNodeCmd)
}

func RunCommandWait(command string) {
	output := exec.Command("/bin/sh", "-c", command)
	output.Stdout = os.Stdout
	output.Stdin = os.Stdin
	err := output.Start()
	output.Wait()
	if err != nil {
		fmt.Println(err.Error())
	}
}

func decommissionNodeRun(cmd *cobra.Command, args []string) {

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

	pmk.DecommissionNode(cfg, nc)

}
