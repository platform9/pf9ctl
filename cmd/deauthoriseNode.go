package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/platform9/pf9ctl/pkg/util"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var deauthNodeCmd = &cobra.Command{
	Use:   "deauthorise-node",
	Short: "Deauthorises this node from the Platform9 control plane",
	Long:  "Removes the host agent package and decommissions this node from the Platform9 control plane. If the node is a part of a single node cluster the cluster will also get deleted.",
	Args: func(deauthNodeCmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errors.New("No parameters are needed")
		}
		return nil
	},
	Run: deauthNodeRun,
}

func init() {
	rootCmd.AddCommand(deauthNodeCmd)
}

func runCommandWait(command string) {
	output := exec.Command("/bin/sh", "-c", command)
	output.Stdout = os.Stdout
	output.Stdin = os.Stdin
	err = output.Start()
	output.Wait()
	if err != nil {
		zap.S().Fatalf("An error has occured ", err)
	}
}

func deauthNodeRun(cmd *cobra.Command, args []string) {

	executor, err := getExecutor(ctx.ProxyURL)

	version, _ := pmk.OpenOSReleaseFile(executor)

	if err != nil {
		zap.S().Fatalf("Error getting OS version")
	}

	var command string

	if strings.Contains(string(version), util.Ubuntu) {
		fmt.Println("Removing packages")
		runCommandWait("sudo dpkg --remove pf9-comms pf9-kube pf9-hostagent pf9-muster")
		fmt.Println("Purging packages")
		runCommandWait("sudo dpkg --purge pf9-comms pf9-kube pf9-hostagent pf9-muster")
		fmt.Println("Removing /etc/pf9 logs")
		runCommandWait("sudo rm -rf /etc/pf9")
		fmt.Println("Removing /opt/pf9 logs")
		runCommandWait("sudo rm -rf /opt/pf9")

	} else {
		command = "sudo yum erase -y pf9-hostagent -y"
		output := exec.Command("/bin/sh", "-c", command)
		output.Stdout = os.Stdout
		output.Stdin = os.Stdin
		err = output.Start()
		output.Wait()
		if err != nil {
			zap.S().Fatalf("An error has occured ", err)
		}

	}

	fmt.Println("Node removed successfully")

}
