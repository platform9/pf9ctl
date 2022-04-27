// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/config"
	"github.com/platform9/pf9ctl/pkg/objects"
	"github.com/platform9/pf9ctl/pkg/util"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// configCmdCreate represents the config command
var (
	configCmdCreate = &cobra.Command{
		Use:   "config",
		Short: "Creates or get the config",
		Long:  `Create or get PF9 controller config used by this CLI`,
	}

	configCmdSet = &cobra.Command{
		Use:   "set",
		Short: "Create a new config",
		Long:  `Create a new config that can be used to query Platform9 controller`,
		Run:   configCmdCreateRun,
	}

	configCmdGet = &cobra.Command{
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

	//cfg objects.UserData
	cfg = &objects.Config{
		Spec: objects.UserData{
			MfaToken: util.MFA,
			OtherData: objects.Other{
				WaitPeriod:    time.Duration(60),
				AllowInsecure: false,
			},
		},
	}
)

func init() {
	rootCmd.AddCommand(configCmdCreate)
	configCmdCreate.AddCommand(configCmdGet)
	configCmdCreate.AddCommand(configCmdSet)

	configCmdSet.Flags().StringVarP(&cfg.Spec.AccountUrl, "account-url", "u", "", "sets account-url")
	configCmdSet.Flags().StringVarP(&cfg.Spec.Username, "username", "e", "", "sets username")
	configCmdSet.Flags().StringVarP(&cfg.Spec.Password, "password", "p", "", "sets password (use 'single quotes' to pass password)")
	configCmdSet.Flags().StringVarP(&cfg.Spec.ProxyURL, "proxy-url", "l", "", "sets proxy URL, can be specified as [<protocol>][<username>:<password>@]<host>:<port>")
	configCmdSet.Flags().StringVarP(&cfg.Spec.Region, "region", "r", "", "sets region")
	configCmdSet.Flags().StringVarP(&cfg.Spec.Tenant, "tenant", "t", "", "sets tenant")
	configCmdSet.Flags().StringVar(&cfg.Spec.MfaToken, "mfa", "", "set MFA token")
}

func configCmdCreateRun(cmd *cobra.Command, args []string) {
	zap.S().Debug("==========Running set config==========")

	var err error
	if err = config.SetProxy(cfg.Spec.ProxyURL); err != nil {
		zap.S().Fatal(color.Red("x "), err)
	}

	if cmd.Flags().Changed("no-prompt") {
		if err = config.ValidateUserCredentials(cfg, objects.NodeConfig{}); err != nil {
			zap.S().Fatal(color.Red("x "), err)
		}

		if err = config.StoreConfig(cfg, util.Pf9DBLoc); err != nil {
			zap.S().Fatal(color.Red("x "), err)
		}

	} else {
		if err = config.GetConfigRecursive(util.Pf9DBLoc, cfg, objects.NodeConfig{}); err != nil {
			zap.S().Fatal(color.Red("x "), err)
		}
	}

	zap.S().Debug("==========Finished running set config==========")
}
