// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/platform9/pf9ctl/pkg/util"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// configCmdCreate represents the config command
var configCmdCreate = &cobra.Command{
	Use:   "config",
	Short: "Create or get config",
	Long:  `Create or get PF9 controller config used by this CLI`,
}

func configCmdCreateRun(cmd *cobra.Command, args []string) {
	zap.S().Debug("==========Running set config==========")
	// invoked the configcreate command from pkg/pmk

	var (
		flag = true
		ctx  pmk.Config
	)

	for flag {
		ctx, _ = pmk.ConfigCmdCreateRun()
		executor, err := getExecutor()
		if err != nil {
			zap.S().Fatalf("Error connecting to host %s", err.Error())
		}
		c, err := pmk.NewClient(ctx.Fqdn, executor, ctx.AllowInsecure, false)
		if err != nil {
			zap.S().Fatalf("Unable to load clients needed for the Cmd. Error: %s", err.Error())
		}

		defer c.Segment.Close()

		zap.S().Debug("==========Validating the User Credentials==========")

		_, err = c.Keystone.GetAuth(
			ctx.Username,
			ctx.Password,
			ctx.Tenant,
		)

		if err != nil {
			zap.S().Debug("Invalid credentials entered (Username/Password/Tenant)")
		} else {
			if err := pmk.StoreConfig(ctx, util.Pf9DBLoc); err != nil {
				zap.S().Errorf("Failed to store config: %s", err.Error())
			}
			flag = false
		}
	}
	if err := pmk.StoreConfig(ctx, util.Pf9DBLoc); err != nil {
		zap.S().Errorf("Failed to store config: %s", err.Error())
	}
	zap.S().Debug("==========Finished running set config==========")
}

var configCmdGet = &cobra.Command{
	Use:   "get",
	Short: "Print stored config",
	Long:  `Print details of the stored config`,
	Run: func(cmd *cobra.Command, args []string) {
		zap.S().Debug("==========Running get config==========")
		_, err := os.Stat(util.Pf9DBLoc)
		if err != nil || os.IsNotExist(err) {
			zap.S().Fatal("Could not load config: ", err)
		}

		file, err := os.Open(util.Pf9DBLoc)
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
		zap.S().Debug("==========Finished running get config==========")
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
