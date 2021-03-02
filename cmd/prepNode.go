// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/platform9/pf9ctl/pkg/util"
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
	ctx, err := pmk.LoadConfig(util.Pf9DBLoc)
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
	// If all pre-requisite checks passed in Check-Node then prep-node
	result, err := pmk.CheckNode(ctx, c)
	if err != nil {
		zap.S().Fatalf("\nPre-requisite check(s) failed %s\n", err.Error())
	}
	if result == pmk.RequiredFail {
		fmt.Println("\nRequired pre-requisite check(s) failed.")
		return
	} else if result == pmk.OptionalFail {
		fmt.Print("\nOptional pre-requisite check(s) failed. Do you want to continue? (y/n) ")
		reader := bufio.NewReader(os.Stdin)
		char, _, _ := reader.ReadRune()
		if char != 'y' {
			return
		}
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
			if user == "" {
				fmt.Printf("Enter username for remote host: ")
				reader := bufio.NewReader(os.Stdin)
				user, _ = reader.ReadString('\n')
				user = strings.TrimSpace(user)
			}
			if sshKey == "" && password == "" {
				var choice int
				fmt.Println("You can choose either password or sshKey")
				fmt.Println("Enter 1 for password and 2 for sshKey")
				fmt.Print("Enter Option : ")
				fmt.Scanf("%d", &choice)
				switch choice {
				case 1:
					fmt.Printf("Enter password for remote host: ")
					passwordBytes, _ := terminal.ReadPassword(0)
					password = string(passwordBytes)
				case 2:
					fmt.Printf("Enter private sshKey: ")
					reader := bufio.NewReader(os.Stdin)
					sshKey, _ = reader.ReadString('\n')
					sshKey = strings.TrimSpace(sshKey)
				default:
					zap.S().Fatalf("Wrong choice please try again")
				}
				fmt.Printf("\n")
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
