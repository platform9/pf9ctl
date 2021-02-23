// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"fmt"
	"io/ioutil"

	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh/terminal"
)

// prepNodeCmd represents the prepNode command
var prepNodeCmd = &cobra.Command{
	Use:   "prep-node",
	Short: "set up prerequisites & prep the node for k8s",
	Long: `Prepare a node to be ready to be added to a Kubernetes cluster. Read more
	at http://pf9.io/cli_clprep.`,
	Run: prepNodeRun,
}

var (
	user       string
	password   string
	sshKey     string
	ips        []string
	floatingIP bool
)

func init() {
	prepNodeCmd.Flags().StringVarP(&user, "user", "u", "", "ssh username for the nodes")
	prepNodeCmd.Flags().StringVarP(&password, "password", "p", "", "ssh password for the nodes")
	prepNodeCmd.Flags().StringVarP(&sshKey, "ssh-key", "s", "", "ssh key file for connecting to the nodes")
	prepNodeCmd.Flags().StringSliceVarP(&ips, "ip", "i", []string{}, "IP address of host to be prepared")
	//prepNodeCmd.Flags().BoolVarP(&floatingIP, "floating-ip", "f", false, "") //Unsupported in first version.

	rootCmd.AddCommand(prepNodeCmd)
}

func prepNodeRun(cmd *cobra.Command, args []string) {
	zap.S().Debug("==========Running prep-node==========")
	ctx, err := pmk.LoadConfig(Pf9DBLoc)
	if err != nil {
		zap.S().Fatalf("Unable to load the config: %s\n", err.Error())
	}
	// TODO: there seems to be a bug, we will need multiple executors one per ip, so at this moment
	// it will only work with one remote host
	executor, err := getExecutor()
	if err != nil {
		zap.S().Fatalf("Error connecting to host %s", err.Error())
	}
	c, err := pmk.NewClient(ctx.Fqdn, executor, ctx.AllowInsecure, false)
	if err != nil {
		zap.S().Fatalf("Unable to load clients needed for the Cmd. Error: %s", err.Error())
	}
	defer c.Segment.Close()

	_, err = pmk.CheckNode(ctx, c)
	if err != nil {
		zap.S().Fatalf("Checknode - Failed: %s\n", err.Error())
	}

	if err := pmk.PrepNode(ctx, c); err != nil {
		c.Segment.SendEvent("Prep Node - Failed", err)
		zap.S().Fatalf("Unable to prep node: %s\n", err.Error())
	}
	zap.S().Debug("==========Finished running prep-node==========")
}

// checkAndValidateRemote check if any of the command line
func checkAndValidateRemote() bool {
	foundRemote := false
	for _, ip := range ips {
		if ip != "localhost" && ip != "127.0.0.1" && ip != "::1" {
			// lets create a remote executor, but before that check if we got user and either of password or ssh-key
			if user == "" || (sshKey == "" && password == "") {
				fmt.Printf("Enter Password: ")
				passwordBytes, _ := terminal.ReadPassword(0)
				password = string(passwordBytes)
			}
			foundRemote = true
			return foundRemote
		}
	}
	zap.S().Debug("Using local executor")
	return foundRemote
}

// getExecutor creates the right Executor
func getExecutor() (cmdexec.Executor, error) {
	if checkAndValidateRemote() {
		var pKey []byte
		var err error
		if sshKey != "" {
			pKey, err = ioutil.ReadFile(sshKey)
			if err != nil {
				zap.S().Fatalf("Unable to read the sshKey %s, %s", sshKey, err.Error())
			}
		}
		return cmdexec.NewRemoteExecutor(ips[0], 22, user, pKey, password)
	}
	zap.S().Debug("Using local executor")
	return cmdexec.LocalExecutor{}, nil
}
