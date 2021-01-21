// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh/terminal"
)

// configCmdCreate represents the config command
var configCmdCreate = &cobra.Command{
	Use:   "config",
	Short: "Create or get config",
	Long:  `Create or get PF9 controller config used by this CLI`,
}

func configCmdCreateRun(cmd *cobra.Command, args []string) {
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

	ctx := pmk.Config{
		Fqdn:          fqdn,
		Username:      username,
		Password:      password,
		Region:        region,
		Tenant:        service,
		WaitPeriod:    WaitPeriod,
		AllowInsecure: false,
	}

	if err := pmk.StoreConfig(ctx, Pf9DBLoc); err != nil {
		zap.S().Errorf("Failed to store config: %s", err.Error())
	}
}

var configCmdGet = &cobra.Command{
	Use:   "get",
	Short: "Print stored config",
	Long:  `Print details of the stored config`,
	Run: func(cmd *cobra.Command, args []string) {
		_, err := os.Stat(Pf9DBLoc)
		if err != nil || os.IsNotExist(err) {
			zap.S().Fatal("Could not load config: ", err)
		}

		file, err := os.Open(Pf9DBLoc)
		if err != nil {
			zap.S().Fatal("Could not load config: ", err)
		}
		defer func() {
			if err = file.Close(); err != nil {
				zap.S().Error(err)
			}
		}()

		data, err := ioutil.ReadAll(file)
		if err != nil {
			zap.S().Fatal("Could not load config: ", err)
		}

		fmt.Printf(string(data))
	},
}

var configCmdSet = &cobra.Command{
	Use:   "set",
	Short: "Create a new config",
	Long:  `Create a new config that can be used to query Platform9 controller`,
	Run:   configCmdCreateRun,
}

func init() {
	rootCmd.AddCommand(configCmdCreate)
	configCmdCreate.AddCommand(configCmdGet)
	configCmdCreate.AddCommand(configCmdSet)
}
