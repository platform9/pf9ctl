package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var removeNodeCmd = &cobra.Command{
	Use:   "deauthorise-node",
	Short: "Deauthorises this node from the Platform9 control plane",
	Long:  "Removes the host agent package and decomissions this node from the Platform9 control plane",
	Args: func(attachNodeCmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errors.New("No parameters are needed")
		}
		return nil
	},
	Run: removeNodeRun,
}

func init() {
	rootCmd.AddCommand(removeNodeCmd)
}

func removeNodeRun(cmd *cobra.Command, args []string) {

	osCheck, err := exec.Command("/bin/sh", "-c", "awk -F= '/^NAME/{print $2}' /etc/os-release").Output()
	if err != nil {
		zap.S().Fatalf("Error getting OS version")
	}

	var command string

	if strings.Contains(string(osCheck), "Ubuntu") {
		command = "sudo apt-get purge pf9-hostagent"

	} else {
		command = "sudo yum erase -y pf9-hostagent"

	}

	output := exec.Command("/bin/sh", "-c", command)
	output.Stdout = os.Stdout
	output.Stdin = os.Stdin
	err = output.Start()
	output.Wait()
	if err != nil {
		zap.S().Fatalf("An error has occured ", err)
	}

	fmt.Println("Node removed successfully")

}
