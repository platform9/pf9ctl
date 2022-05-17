package cmd

import "github.com/spf13/cobra"

var (
	cloudProviderCmd = &cobra.Command{
		Use:   "provider",
		Short: "Cloud provider check commands",
		Long:  `Cloud provider check commands`,
	}
)

func init() {
	rootCmd.AddCommand(cloudProviderCmd)
}
