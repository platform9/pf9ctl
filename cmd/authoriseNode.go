package cmd

import (
	"errors"
	"fmt"

	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/platform9/pf9ctl/pkg/util"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var authNodeCmd = &cobra.Command{
	Use:   "authorize-node",
	Short: "Authorizes this node from the Platform9 control plane",
	Long:  "Authorizes this node",
	Args: func(deauthNodeCmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errors.New("No parameters are needed")
		}
		return nil
	},
	Run: authNodeRun,
}

func init() {
	rootCmd.AddCommand(authNodeCmd)
}

func authNodeRun(cmd *cobra.Command, args []string) {

	ctx, err = pmk.LoadConfig(util.Pf9DBLoc)

	if err != nil {
		zap.S().Fatalf("Error loading config", err)
	}

	executor, err := getExecutor(ctx.ProxyURL)

	c, err = pmk.NewClient(ctx.Fqdn, executor, ctx.AllowInsecure, false)

	if err != nil {
		zap.S().Fatalf("Error getting OS version")
	}

	auth, err := c.Keystone.GetAuth(ctx.Username, ctx.Password, ctx.Tenant, ctx.MfaToken)
	if err != nil {
		zap.S().Debug("Failed to get keystone %s", err.Error())
	}

	var nodeIPs []string
	nodeIPs = append(nodeIPs, getIp().String())

	token := auth.Token
	nodeUuids, _ := hostId(c.Executor, ctx.Fqdn, token, nodeIPs)

	if len(nodeUuids) == 0 {
		zap.S().Fatalf("Could not find the node. Check if the node associated with this account")
	}

	err1 := c.Qbert.AuthoriseNode(nodeUuids[0], token)

	if err1 != nil {
		zap.S().Fatalf("Error authorising node ", err1.Error())
	}

	fmt.Println("Finished authorizing node")

}
