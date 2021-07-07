// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/log"
	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/platform9/pf9ctl/pkg/supportBundle"
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
	Args: func(prepNodeCmd *cobra.Command, args []string) error {
		if prepNodeCmd.Flags().Changed("disableSwapOff") {
			util.SwapOffDisabled = true
		}
		return nil
	},
}

var (
	user           string
	password       string
	sshKey         string
	ips            []string
	skipChecks     bool
	disableSwapOff bool
)

func init() {
	prepNodeCmd.Flags().StringVarP(&user, "user", "u", "", "ssh username for the nodes")
	prepNodeCmd.Flags().StringVarP(&password, "password", "p", "", "ssh password for the nodes (use 'single quotes' to pass password)")
	prepNodeCmd.Flags().StringVarP(&sshKey, "ssh-key", "s", "", "ssh key file for connecting to the nodes")
	prepNodeCmd.Flags().StringSliceVarP(&ips, "ip", "i", []string{}, "IP address of host to be prepared")
	prepNodeCmd.Flags().BoolVarP(&skipChecks, "skipChecks", "c", false, "Will skip optional checks if true")
	prepNodeCmd.Flags().BoolVarP(&disableSwapOff, "disableSwapOff", "d", false, "Will skip swapoff")
	prepNodeCmd.Flags().MarkHidden("disableSwapOff")

	rootCmd.AddCommand(prepNodeCmd)
}

func prepNodeRun(cmd *cobra.Command, args []string) {
	zap.S().Debug("==========Running prep-node==========")
	// This flag is used to loop back if user enters invalid credentials during config set.
	credentialFlag = true
	// To bail out if loop runs recursively more than thrice
	pmk.LoopCounter = 0

	for credentialFlag {
		ctx, err = pmk.LoadConfig(util.Pf9DBLoc)
		if err != nil {
			zap.S().Fatalf("Unable to load the config: %s\n", err.Error())
		}

		executor, err := getExecutor(ctx.ProxyURL)
		if err != nil {
			zap.S().Debug("Error connecting to host %s", err.Error())
			zap.S().Fatalf(" Invalid (Username/Password/IP), use 'single quotes' to pass password")
		}

		c, err = pmk.NewClient(ctx.Fqdn, executor, ctx.AllowInsecure, false)
		if err != nil {
			zap.S().Fatalf("Unable to load clients needed for the Cmd. Error: %s", err.Error())
		}

		defer c.Segment.Close()

		// Validate the user credentials entered during config set and will bail out if invalid
		if err := validateUserCredentials(ctx, c); err != nil {
			//Clearing the invalid config entered. So that it will ask for new information again.
			clearContext(&pmk.Context)
			//Check if no or invalid config exists, then bail out if asked for correct config for maxLoop times.
			err = configValidation(RegionInvalid, pmk.LoopCounter)
		} else {
			// We will store the set config if its set for first time using check-node
			if pmk.IsNewConfig {
				if err := pmk.StoreConfig(ctx, util.Pf9DBLoc); err != nil {
					zap.S().Errorf("Failed to store config: %s", err.Error())
				} else {
					pmk.IsNewConfig = false
				}
			}
			credentialFlag = false
		}
	}
	// If all pre-requisite checks passed in Check-Node then prep-node
	result, err := pmk.CheckNode(ctx, c)
	if err != nil {
		// Uploads pf9cli log bundle if pre-requisite checks fails
		errbundle := supportBundle.SupportBundleUpload(ctx, c)
		if errbundle != nil {
			zap.S().Debugf("Unable to upload supportbundle to s3 bucket %s", errbundle.Error())
		}
		zap.S().Fatalf("\nPre-requisite check(s) failed %s\n", err.Error())
	}

	if result == pmk.RequiredFail {
		fmt.Println(color.Red("x ") + "Required pre-requisite check(s) failed.")
		return
	} else if !skipChecks {
		if result == pmk.OptionalFail {
			fmt.Print("\nOptional pre-requisite check(s) failed. Do you want to continue? (y/n) ")
			reader := bufio.NewReader(os.Stdin)
			char, _, _ := reader.ReadRune()
			if char != 'y' {
				return
			}
		}
	}

	if err := pmk.PrepNode(ctx, c); err != nil {
		fmt.Printf("\nFailed to prepare node. See %s or use --verbose for logs\n", log.GetLogLocation(util.Pf9Log))

		// Uploads pf9cli log bundle if prepnode failed to get prepared
		errbundle := supportBundle.SupportBundleUpload(ctx, c)
		if errbundle != nil {
			zap.S().Debugf("Unable to upload supportbundle to s3 bucket %s", errbundle.Error())
		}

		zap.S().Debugf("Unable to prep node: %s\n", err.Error())
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
			supportBundle.RemoteBundle = true
			pmk.IsRemoteExecutor = true
			return foundRemote
		}
	}
	zap.S().Debug("Using local executor")
	return foundRemote
}

// getExecutor creates the right Executor
func getExecutor(proxyURL string) (cmdexec.Executor, error) {
	if checkAndValidateRemote() {
		var pKey []byte
		var err error
		if sshKey != "" {
			pKey, err = ioutil.ReadFile(sshKey)
			if err != nil {
				zap.S().Fatalf("Unable to read the sshKey %s, %s", sshKey, err.Error())
			}
		}
		return cmdexec.NewRemoteExecutor(ips[0], 22, user, pKey, password, proxyURL)
	}
	zap.S().Debug("Using local executor")
	return cmdexec.LocalExecutor{ProxyUrl: proxyURL}, nil
}
