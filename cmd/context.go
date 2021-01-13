// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/platform9/pf9ctl/pkg/util"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh/terminal"
)

// contextCmdCreate represents the context command
var contextCmdCreate = &cobra.Command{
	Use:   "context",
	Short: "Create a new context",
	Long:  `Create a new context that can be used to query Platform9 controller`,
	Run:   contextCmdCreateRun,
}

func contextCmdCreateRun(cmd *cobra.Command, args []string) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("Platform9 Account URL: ")
	fqdn, _ := reader.ReadString('\n')
	fqdn = strings.TrimSuffix(fqdn, "\n")

	fmt.Printf("Username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSuffix(username, "\n")

	fmt.Printf("Password: ")
	passwordBytes, _ := terminal.ReadPassword(0)
	password := string(passwordBytes)

	fmt.Printf("\nRegion [RegionOne]: ")
	region, _ := reader.ReadString('\n')
	region = strings.TrimSuffix(region, "\n")

	fmt.Printf("Tenant [service]: ")
	service, _ := reader.ReadString('\n')
	service = strings.TrimSuffix(service, "\n")

	if region == "" {
		region = "RegionOne"
	}

	if service == "" {
		service = "service"
	}

	ctx := pmk.Context{
		Fqdn:          fqdn,
		Username:      username,
		Password:      password,
		Region:        region,
		Tenant:        service,
		WaitPeriod:    WaitPeriod,
		AllowInsecure: false,
	}

	if err := pmk.StoreContext(ctx, Pf9DBLoc); err != nil {
		zap.S().Errorf("Failed to store context: %s", err.Error())
	}
}

var contextCmdGet = &cobra.Command{
	Use:   "get",
	Short: "Print stored context",
	Long:  `Print details of the stored context`,
	Run: func(cmd *cobra.Command, args []string) {
		data, err := util.ReadFile(Pf9DBLoc)
		if err != nil {
			fmt.Printf("No context found: %s\n", err)
		}

		fmt.Printf(string(data))
	},
}

func init() {
	rootCmd.AddCommand(contextCmdCreate)
	contextCmdCreate.AddCommand(contextCmdGet)
}
