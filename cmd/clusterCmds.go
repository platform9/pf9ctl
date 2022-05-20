package cmd

import "github.com/spf13/cobra"

var (
	clusterCmd = &cobra.Command{
		Use:   "cluster",
		Short: "Platform9 Managed Kubernetes cluster operations",
		Long:  `Platform9 Managed Kubernetes cluster operations`,
	}
)

func init() {
	rootCmd.AddCommand(clusterCmd)
}
