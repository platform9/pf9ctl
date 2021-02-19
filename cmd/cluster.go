// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// clusterCmdGet represents the cluster get command
var clusterCmdGet = &cobra.Command{
	Use:   "cluster",
	Short: "Display one or many clusters",
	Long: `Query your controller using the current config and list
	 the clusters`,
	Run: func(cmd *cobra.Command, args []string) {
		zap.S().Info("Get cluster called")
	},
}

// clusterCmdCreate represents Create cluster command
var clusterCmdCreate = &cobra.Command{
	Use:   "cluster",
	Short: "Create a kubernetes cluster",
	Long:  `Create a cluster and add one or more nodes to it.`,
	Run: func(cmd *cobra.Command, args []string) {
		zap.S().Info("Create cluster called")
	},
}

/*
// This initialization is using create and get commands which are not in use for now.
func init() {

	//createCmd.AddCommand(clusterCmdCreate)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// clusterCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// clusterCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	//getCmd.AddCommand(clusterCmdGet)
}*/
