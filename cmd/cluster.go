// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"fmt"

	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	clusterHeadless   bool
	clusterConfigPath string
)

// clusterCmdCreate represents Create cluster command
var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Manage PMK clusters",
	Long:  "Create and delete PMK clusters locally",
}

// clusterCmdGet represents the cluster get command
var clusterCmdGet = &cobra.Command{
	Use:   "get",
	Short: "Display one or many clusters",
	Long: `Query your controller using the current config and list
	 the clusters`,
	Run: func(cmd *cobra.Command, args []string) {
		zap.S().Info("Get cluster called")
	},
}

// clusterCmdCreate represents Create cluster command
var clusterCmdCreate = &cobra.Command{
	Use:   "create",
	Short: "Create a kubernetes cluster",
	Long:  `Create a cluster and add one or more nodes to it.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		zap.S().Info("Create cluster called")
		if !clusterHeadless {
			// Do nothing
			return nil
		}
		err := createHeadlessCluster(clusterConfigPath)
		return err
	},
}

func init() {
	rootCmd.AddCommand(clusterCmd)
	clusterCmd.AddCommand(clusterCmdGet)
	clusterCmd.AddCommand(clusterCmdCreate)
	clusterCmdCreate.Flags().BoolVar(&clusterHeadless, "headless", false, "Create headless clusters; will not talk to the DU")
	clusterCmdCreate.Flags().StringVar(&clusterConfigPath, "config", "/etc/pf9/pf9ctl.yaml", "Path to headless cluster config file")
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

func createHeadlessCluster(configFile string) error {
	viper.SetConfigFile(configFile)

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %s", err)
	}
	pf9KubePath := viper.GetString("pf9KubePath")
	configTarPath := viper.GetString("configTarPath")
	masterNodeList := viper.GetStringSlice("masterNodeList")
	workerNodeList := viper.GetStringSlice("workerNodeList")
	username := viper.GetString("username")
	privKeyPath := viper.GetString("privKeyPath")
	password := viper.GetString("password")

	err = pmk.CreateHeadlessCluster(pf9KubePath, configTarPath, masterNodeList, workerNodeList, username, privKeyPath, password)
	if err != nil {
		return fmt.Errorf("failed to create headless cluster: %s", err)
	}
	return nil
}
