// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// supportBundleCmd represents the supportBundle command
var supportBundleCmd = &cobra.Command{
	Use:   "bundle",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		zap.S().Info("Support bundle called")
	},
}

/*
This initialization is using create commands which is not in use for now.
func init() {
	//createCmd.AddCommand(supportBundleCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// supportBundleCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// supportBundleCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}*/
