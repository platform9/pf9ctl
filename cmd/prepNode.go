// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"github.com/platform9/pf9ctl/pkg/constants"
	"github.com/platform9/pf9ctl/pkg/log"
	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/platform9/pf9ctl/pkg/pmk/clients"
	"github.com/spf13/cobra"
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
	prepNodeCmd.Flags().StringVarP(&sshKey, "ssh-key", "s", "", "ssh key for connecting to the nodes")
	prepNodeCmd.Flags().StringSliceVarP(&ips, "ips", "i", []string{}, "ips of host to be prepared")
	prepNodeCmd.Flags().BoolVarP(&floatingIP, "floating-ip", "f", false, "")

	rootCmd.AddCommand(prepNodeCmd)
}

func prepNodeRun(cmd *cobra.Command, args []string) {

	ctx, err := pmk.LoadContext(constants.Pf9DBLoc)
	if err != nil {
		log.Fatalf("Unable to load the context: %s\n", err.Error())
	}

	c, err := clients.New(ctx.Fqdn, ctx.Proxy)
	if err != nil {
		log.Fatalf("Unable to load clients needed for the Cmd. Error: %s", err.Error())
	}

	if err := pmk.PrepNode(ctx, c, user, password, sshKey, ips); err != nil {
		c.Segment.SendEvent("Prep Node - Failed", err)
		log.Fatalf("Unable to prep node: %s\n", err.Error())
	}
}
