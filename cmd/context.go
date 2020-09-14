// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/prometheus/common/log"
	"github.com/spf13/cobra"
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
	password, _ := reader.ReadString('\n')
	password = strings.TrimSuffix(password, "\n")

	fmt.Printf("Region [RegionOne]: ")
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
	}

	if err := pmk.StoreContext(ctx, pmk.Pf9DBLoc); err != nil {
		log.Errorf("Failed: %s", err.Error())
	}
}

var contextCmdGet = &cobra.Command{
	Use:   "context",
	Short: "List stored context/s",
	Long:  `List stored contexts or details about a specific context`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("context get called")
	},
}

func init() {
	createCmd.AddCommand(contextCmdCreate)
	getCmd.AddCommand(contextCmdGet)
}
