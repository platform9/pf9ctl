// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/platform9/pf9ctl/pkg/color"
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

var (
	ctx pmk.Config
	err error
	c   pmk.Client
	// This flag is used to loop back if user enters invalid credentials during config set.
	credentialFlag bool
)

const MaxLoopNoConfig = 3

func configCmdCreateRun(cmd *cobra.Command, args []string) {
	zap.S().Debug("==========Running set config==========")

	credentialFlag = true

	for credentialFlag {
		// invoked the configcreate command from pkg/pmk
		ctx, _ = pmk.ConfigCmdCreateRun()

		executor, err := getExecutor()
		if err != nil {
			zap.S().Fatalf("Error connecting to host %s", err.Error())
		}

		c, err = pmk.NewClient(ctx.Fqdn, executor, ctx.AllowInsecure, false)
		if err != nil {
			zap.S().Fatalf("Unable to load clients needed for the Cmd. Error: %s", err.Error())
		}

		// Validate the user credentials entered during config set and will bail out if invalid

		if err := validateUserCredentials(ctx, c); err != nil {
			//zap.S().Fatalf("Invalid credentials (Username/ Password/ Account), run 'pf9ctl config set' with correct credentials.")
			zap.S().Info("Invalid credentials entered (Username/Password/Tenant)")

		} else {
			credentialFlag = false
		}
	}

	defer c.Segment.Close()

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

// This function will validate the user credentials entered during config set and bail out if invalid
func validateUserCredentials(pmk.Config, pmk.Client) error {

	_, err = c.Keystone.GetAuth(
		ctx.Username,
		ctx.Password,
		ctx.Tenant,
	)
	return err
}

func configValidation(int) error {

	if pmk.LoopCounter <= MaxLoopNoConfig-1 {
		if !pmk.OldConfigExist {
			zap.S().Debug("Invalid credentials entered (Username/Password/Tenant)")
		} else {
			zap.S().Debug("Invalid credentials found (Username/Password/Tenant)")
		}
	}
	// If existing initial config is Invalid
	if (pmk.LoopCounter == 0) && (pmk.OldConfigExist) {
		pmk.InvalidExistingConfig = true
		pmk.LoopCounter += 1
	} else {
		// If user enteres invalid credentials during new config pormpt.
		pmk.LoopCounter += 1
	}

	// If any invalid credentials extered multiple times in new config prompt then to bail out the recursive loop (thrice)
	if pmk.LoopCounter >= MaxLoopNoConfig && !(pmk.InvalidExistingConfig) {
		fmt.Println(color.Red("x ") + "Invalid credentials entered (Username/Password/Tenant)")
		zap.S().Fatalf("Invalid credentials entered multiple times (Username/Password/Tenant)")
	} else if pmk.LoopCounter >= MaxLoopNoConfig+1 && pmk.InvalidExistingConfig {
		fmt.Println(color.Red("x ") + "Invalid credentials entered (Username/Password/Tenant)")
		zap.S().Fatalf("Invalid credentials entered multiple times (Username/Password/Tenant)")
	}
	return nil
}
