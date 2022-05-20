// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"errors"

	"github.com/platform9/pf9ctl/pkg/config"
	"github.com/platform9/pf9ctl/pkg/util"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	configCreate = &cobra.Command{
		Use:   "create",
		Short: "Creates the spec based config",
		Long:  `Creates the spec based config which will be used by this CLI`,
		Run:   configCreateRun,
		PreRunE: func(configCreate *cobra.Command, args []string) error {
			if configCreate.Flags().Changed("output") {
				if output == "JSON" {
					config.JsonFileType = true
				} else if output == "YAML" {
					config.JsonFileType = false
				} else {
					return errors.New("Supported file extentions are YAML nad JSON")
				}
			}
			return nil
		},
	}
)

var (
	kind          string
	output        string
	fileName      string
	configFileLoc string
)

func init() {
	configCmdCreate.AddCommand(configCreate)
	configCreate.Flags().StringVar(&kind, "kind", "", "Specify the type of config")
	configCreate.Flags().StringVar(&output, "output", "YAML", "Specify the extention type of config file")
	configCreate.Flags().StringVar(&util.ConfigFileName, "file-name", "", "Specify the config file name")
	configCreate.Flags().StringVar(&util.ConfigFileLoc, "location", "", "Specify location of config file to store")
	configCreate.MarkFlagRequired("kind")

}

func configCreateRun(cmd *cobra.Command, args []string) {
	if kind == "user-config" {
		config.CreateUserConfig()
	} else if kind == "node" {
		config.CreateNodeConfig()
	} else {
		zap.S().Fatal("Please make sure kind is user-config/node")
	}
}
