// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a resource",
	Long: `Use the create command to create cluster, config, support bundle and
	other resources`,
	Run: func(cmd *cobra.Command, args []string) {
		zap.S().Info("Create called")
	},
}

/*
This initialization of create command to root isnot in use for now.
func init() {
	//rootCmd.AddCommand(createCmd)
}*/
