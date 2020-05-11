// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// prepNodeCmd represents the prepNode command
var prepNodeCmd = &cobra.Command{
	Use:   "prep-node",
	Short: "set up prerequisites & prep the node for k8s",
	Long: `Prepare a node to be ready to be added to a Kubernetes cluster. Read more
	at http://pf9.io/cli_clprep.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("prepNode called")
	},
}

func init() {
	rootCmd.AddCommand(prepNodeCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// prepNodeCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// prepNodeCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
