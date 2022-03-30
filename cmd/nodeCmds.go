package cmd

import "github.com/spf13/cobra"

var (
	nodeCmd = &cobra.Command{
		Use:   "node",
		Short: "Configure Nodes for Platform9 management planes",
		Long:  `Configure Nodes for Platform9 management planes`,
	}
)

func init() {
	rootCmd.AddCommand(nodeCmd)
}
