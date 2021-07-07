// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"fmt"

	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/log"
	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/platform9/pf9ctl/pkg/supportBundle"
	"github.com/platform9/pf9ctl/pkg/util"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var checkNodeCmd = &cobra.Command{
	Use:   "check-node",
	Short: "Check prerequisites for k8s",
	Long: `Check if a node satisfies prerequisites to be ready to be added to a Kubernetes cluster. Read more
	at https://platform9.com/blog/support/managed-container-cloud-requirements-checklist/`,
	Run: checkNodeRun,
}

func init() {
	checkNodeCmd.Flags().StringVarP(&user, "user", "u", "", "ssh username for the nodes")
	checkNodeCmd.Flags().StringVarP(&password, "password", "p", "", "ssh password for the nodes (use 'single quotes' to pass password)")
	checkNodeCmd.Flags().StringVarP(&sshKey, "ssh-key", "s", "", "ssh key file for connecting to the nodes")
	checkNodeCmd.Flags().StringSliceVarP(&ips, "ip", "i", []string{}, "IP address of host to be prepared")
	//checkNodeCmd.Flags().BoolVarP(&floatingIP, "floating-ip", "f", false, "") //Unsupported in first version.

	rootCmd.AddCommand(checkNodeCmd)
}

func checkNodeRun(cmd *cobra.Command, args []string) {
	zap.S().Debug("==========Running check-node==========")
	// This flag is used to loop back if user enters invalid credentials during config set.
	credentialFlag = true
	// To bail out if loop runs recursively more than thrice
	pmk.LoopCounter = 0

	for credentialFlag {

		ctx, err = pmk.LoadConfig(util.Pf9DBLoc)
		if err != nil {
			zap.S().Fatalf("Unable to load the context: %s\n", err.Error())
		}

		executor, err := getExecutor(ctx.ProxyURL)
		if err != nil {
			zap.S().Fatalf(" Invalid (Username/Password/IP), use 'single quotes' to pass password")
			zap.S().Debug("Error connecting to host %s", err.Error())
		}

		c, err = pmk.NewClient(ctx.Fqdn, executor, ctx.AllowInsecure, false)
		if err != nil {
			zap.S().Fatalf("Unable to load clients needed for the Cmd. Error: %s", err.Error())
		}

		// Validate the user credentials entered during config set and will loop back again if invalid
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

	defer c.Segment.Close()

	result, err := pmk.CheckNode(ctx, c)
	if err != nil {
		// Uploads pf9cli log bundle if checknode fails
		errbundle := supportBundle.SupportBundleUpload(ctx, c)
		if errbundle != nil {
			zap.S().Debugf("Unable to upload supportbundle to s3 bucket %s", errbundle.Error())
		}
		zap.S().Fatalf("Unable to perform pre-requisite checks on this node: %s", err.Error())
	}

	if result == pmk.RequiredFail {
		fmt.Printf(color.Red("x ")+"Required pre-requisite check(s) failed. See %s or use --verbose for logs \n", log.GetLogLocation(util.Pf9Log))
	} else if result == pmk.OptionalFail {
		fmt.Printf("\nOptional pre-requisite check(s) failed. See %s or use --verbose for logs \n", log.GetLogLocation(util.Pf9Log))
	}

	zap.S().Debug("==========Finished running check-node==========")
}
