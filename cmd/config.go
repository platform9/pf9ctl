// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"time"

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
	// This flag is true when we set config through ./pf9ctl config set
	SetConfig             bool
	SetConfigByParameters bool
	// This flag is set true when only region is found/entered is invalid.
	RegionInvalid bool
)

const MaxLoopNoConfig = 3

//This function clears the context if it is invalid. Before storing it.
func clearContext(v interface{}) {
	p := reflect.ValueOf(v).Elem()
	p.Set(reflect.Zero(p.Type()))
}

func configCmdCreateRun(cmd *cobra.Command, args []string) {
	zap.S().Debug("==========Running set config==========")

	credentialFlag = true
	SetConfig = true
	// To bail out if loop runs recursively more than thrice
	pmk.LoopCounter = 0

	pmk.Context.Fqdn = account_url
	pmk.Context.Username = username
	pmk.Context.Password = Password
	pmk.Context.Region = region
	pmk.Context.Tenant = tenant
	pmk.Context.WaitPeriod = time.Duration(60)
	pmk.Context.AllowInsecure = false

	for credentialFlag {
		// invoked the configcreate command from pkg/pmk
		ctx, _ = pmk.ConfigCmdCreateRun()

		executor, err := getExecutor(ctx.ProxyURL)
		if err != nil {
			zap.S().Fatalf("Error connecting to host %s", err.Error())
		}

		c, err = pmk.NewClient(ctx.Fqdn, executor, ctx.AllowInsecure, false)
		if err != nil {
			zap.S().Fatalf("Unable to load clients needed for the Cmd. Error: %s", err.Error())
		}

		// Validate the user credentials entered during config set and will bail out if invalid

		if err := validateUserCredentials(ctx, c); err != nil {

			if SetConfigByParameters {
				if RegionInvalid {
					zap.S().Fatalf("Invalid credentials entered (Region)")
				} else {
					zap.S().Fatalf("Invalid credentials entered (Platform9 Account URL/Username/Password/Region/Tenant), use 'single quotes' to pass password")
				}
				credentialFlag = false
			} else {
				//Clearing the invalid config entered. So that it will ask for new information again.
				clearContext(&pmk.Context)
				//Check if no or invalid config exists, then bail out if asked for correct config for maxLoop times.
				err = configValidation(RegionInvalid, pmk.LoopCounter)
			}

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
	Args: func(configCmdSet *cobra.Command, args []string) error {
		if configCmdSet.Flags().Changed("account_url") || configCmdSet.Flags().Changed("username") || configCmdSet.Flags().Changed("password") || configCmdSet.Flags().Changed("region") || configCmdSet.Flags().Changed("tenant") {
			SetConfigByParameters = true
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmdCreate)
	configCmdCreate.AddCommand(configCmdGet)
	configCmdCreate.AddCommand(configCmdSet)
}

var (
	account_url string
	username    string
	Password    string
	region      string
	tenant      string
)

func init() {
	configCmdSet.Flags().StringVarP(&account_url, "account_url", "u", "", "sets account_url")
	configCmdSet.Flags().StringVarP(&username, "username", "e", "", "sets username")
	configCmdSet.Flags().StringVarP(&Password, "password", "p", "", "sets password (use 'single quotes' to pass password)")
	configCmdSet.Flags().StringVarP(&region, "region", "r", "", "sets region")
	configCmdSet.Flags().StringVarP(&tenant, "tenant", "t", "", "sets tenant")
}

// This function will validate the user credentials entered during config set and bail out if invalid
func validateUserCredentials(pmk.Config, pmk.Client) error {

	auth, err := c.Keystone.GetAuth(
		ctx.Username,
		ctx.Password,
		ctx.Tenant,
	)

	if err != nil {
		RegionInvalid = false
		return err
	}

	// To validate region.
	endpointURL, err1 := pmk.FetchRegionFQDN(ctx, auth)
	if endpointURL == "" || err1 != nil {
		RegionInvalid = true
		zap.S().Debug("Invalid Region")
		return errors.New("Invalid Region")
	}

	return nil
}

func configValidation(bool, int) error {

	if pmk.LoopCounter <= MaxLoopNoConfig-1 {

		//Check if we are setting config through pf9ctl config set command.
		if !SetConfig {
			// If Oldconfig exists and invalid credentials entered during config prompt
			if pmk.OldConfigExist {
				if pmk.InvalidExistingConfig {
					// If user enters invalid credentials during prompt of config (due to invalid config found after config loading).
					if RegionInvalid {
						fmt.Println("\n" + color.Red("x ") + "Invalid Region entered")
						zap.S().Debug("Invalid Region entered")
					} else {
						fmt.Println("\n" + color.Red("x ") + "Invalid credentials entered (Platform9 Account URL/Username/Password/Region/Tenant)")
						zap.S().Debug("Invalid credentials entered (Platform9 Account URL/Username/Password/Region/Tenant)")
					}

				} else if pmk.OldConfigExist && pmk.LoopCounter == 0 {
					// If invalid credentials are found during config loading
					if RegionInvalid {
						fmt.Println("\n" + color.Red("x ") + "Invalid Region found")
						zap.S().Debug("Invalid Region found")
					} else {
						fmt.Println("\n" + color.Red("x ") + "Invalid credentials found (Platform9 Account URL/Username/Password/Region/Tenant)")
						zap.S().Debug("Invalid credentials found (Platform9 Account URL/Username/Password/Region/Tenant)")
					}
				}

			} else {
				// If user enters invalid credentials during new config promput (due to config not found)
				if RegionInvalid {
					fmt.Println("\n" + color.Red("x ") + "Invalid Region entered")
					zap.S().Debug("Invalid Region entered")
				} else {
					fmt.Println("\n" + color.Red("x ") + "Invalid credentials entered (Platform9 Account URL/Username/Password/Region/Tenant)")
					zap.S().Debug("Invalid credentials entered (Platform9 Account URL/Username/Password/Region/Tenant)")
				}

			}
		} else {
			// If user enters invalid credentials during config set through "pf9ctl config set"
			if RegionInvalid {
				fmt.Println("\n" + color.Red("x ") + "Invalid Region entered")
				zap.S().Debug("Invalid Region entered")
			} else {
				fmt.Println("\n" + color.Red("x ") + "Invalid credentials entered (Platform9 Account URL/Username/Password/Region/Tenant)")
				zap.S().Debug("Invalid credentials entered (Platform9 Account URL/Username/Password/Region/Tenant)")
			}

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
		zap.S().Fatalf("Invalid credentials entered multiple times (Platform9 Account URL/Username/Password/Region/Tenant)")
	} else if pmk.LoopCounter >= MaxLoopNoConfig+1 && pmk.InvalidExistingConfig {
		if RegionInvalid {
			fmt.Println(color.Red("x ") + "Invalid Region entered")
		} else {
			fmt.Println(color.Red("x ") + "Invalid credentials entered (Platform9 Account URL/Username/Password/Region/Tenant)")
		}
		zap.S().Fatalf("Invalid credentials entered multiple times (Platform9 Account URL/Username/Password/Region/Tenant)")
	}
	return nil
}
