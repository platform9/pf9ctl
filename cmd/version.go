// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"fmt"

	"github.com/platform9/pf9ctl/pkg/util"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Current version of CLI being used",
	Long:  "Gives the current pf9ctl version",
	Run: func(cmd *cobra.Command, args []string) {
		zap.S().Debug("Version called")
		//Prints the current version of pf9ctl being used.
		fmt.Println(util.Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// versionCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// versionCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
