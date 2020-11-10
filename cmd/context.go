// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/spf13/cobra"
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
	password:= string(passwordBytes)
	

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
		Fqdn:     fqdn,
		Username: username,
		Password: password,
		Region:   region,
		Tenant:   service,
		WaitPeriod: WaitPeriod,
		AllowInsecure: false,
	}

	if err := pmk.StoreContext(ctx, Pf9DBLoc); err != nil {
		zap.S().Errorf("Failed to store context: %s", err.Error())
	}
}

var contextCmdGet = &cobra.Command{
	Use:   "context",
	Short: "List stored context/s",
	Long:  `List stored contexts or details about a specific context`,
	Run: func(cmd *cobra.Command, args []string) {
		zap.S().Info("Get context called")
	},
}

func init() {
	rootCmd.AddCommand(contextCmdCreate)
	rootCmd.AddCommand(contextCmdGet)
}
