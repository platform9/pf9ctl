// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"go.uber.org/zap"
	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/spf13/cobra"
	"io/ioutil"
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
	prepNodeCmd.Flags().StringSliceVarP(&ips, "ips", "i", []string{}, "ips of host to be prepared")
	prepNodeCmd.Flags().BoolVarP(&floatingIP, "floating-ip", "f", false, "")

	rootCmd.AddCommand(prepNodeCmd)
}


func prepNodeRun(cmd *cobra.Command, args []string) {

	ctx, err := pmk.LoadContext(Pf9DBLoc)
	if err != nil {
		zap.S().Fatalf("Unable to load the context: %s\n", err.Error())
	}
	// TODO: there seems to be a bug, we will need multiple executors one per ip, so at this moment
	// it will only work with one remote host
	executor, err := getExecutor()
	if err != nil {
		zap.S().Fatalf("Error connecting to host %s",err.Error())
	}
	c, err := pmk.NewClient(ctx.Fqdn, executor)
	if err != nil {
		zap.S().Fatalf("Unable to load clients needed for the Cmd. Error: %s", err.Error())
	}

	if err := pmk.PrepNode(ctx, c); err != nil {
		c.Segment.SendEvent("Prep Node - Failed", err)
		zap.S().Fatalf("Unable to prep node: %s\n", err.Error())
	}
}

// checkAndValidateRemote check if any of the command line 
func checkAndValidateRemote() bool {
	foundRemote := false
	for _, ip := range ips {
		if ip != "localhost" && ip != "127.0.0.1" && ip != "::1" {
			// lets create a remote executor, but before that check if we got user and either of password or ssh-key
			if user =="" || (sshKey == "" && password == "") {
				zap.S().Fatalf("please provider 'user' and one of 'password' or ''ssh-key'")
			}
			foundRemote = true
			return foundRemote
		}
	}
	zap.S().Info("Using local exeuctor")
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
				zap.S().Fatalf("Unale to read the sshKey %s, %s", sshKey, err.Error())
			}
		}
		return cmdexec.NewRemoteExecutor(ips[0], 22, user, pKey, password)
 	}
	zap.S().Info("Using local exeuctor")
	return cmdexec.LocalExecutor{}, nil
}
