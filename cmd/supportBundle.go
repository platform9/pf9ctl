// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"go.uber.org/zap"
	"github.com/spf13/cobra"
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

func init() {
	createCmd.AddCommand(supportBundleCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// supportBundleCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// supportBundleCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
