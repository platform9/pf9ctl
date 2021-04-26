package cmd

import (
	"github.com/spf13/cobra"
)

var attachNodeCmd = &cobra.Command{
	Use:   "attach-node",
	Short: "attaches node to kubernetes cluster",
	Long:  "attaches node to kubernetes cluster",
	Run:   attachNodeRun,
}

var (
	masterIp []string
	workerIp []string
)

func init() {
	prepNodeCmd.Flags().StringSliceVarP(&masterIp, "master-ip", "m", []string{}, "master node ip address")
	prepNodeCmd.Flags().StringSliceVarP(&workerIp, "worker-ip", "w", []string{}, "worker node ip address")

	rootCmd.AddCommand(attachNodeCmd)
}

func attachNodeRun(cmd *cobra.Command, args []string) {

}
