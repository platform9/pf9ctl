// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"fmt"

	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/platform9/pf9ctl/pkg/supportBundle"
	"github.com/platform9/pf9ctl/pkg/util"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// supportBundleCmd represents the supportBundle command
var supportBundleCmd = &cobra.Command{
	Use:   "bundle",
	Short: "Gathers support bundle and uploads to S3",
	Long:  `Gathers support bundle that includes logs for pf9 services and pf9ctl, uploads to S3 `,
	Run:   supportBundleUpload,
}

//This initialization is using create commands which is not in use for now.
func init() {
	supportBundleCmd.Flags().StringVarP(&user, "user", "u", "", "ssh username for the nodes")
	supportBundleCmd.Flags().StringVarP(&password, "password", "p", "", "ssh password for the nodes (use 'single quotes' to pass password)")
	supportBundleCmd.Flags().StringVarP(&sshKey, "ssh-key", "s", "", "ssh key file for connecting to the nodes")
	supportBundleCmd.Flags().StringSliceVarP(&ips, "ip", "i", []string{}, "IP address of host to be prepared")

	rootCmd.AddCommand(supportBundleCmd)
}

func supportBundleUpload(cmd *cobra.Command, args []string) {
	zap.S().Debug("==========Running supportBundleUpload==========")
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
			zap.S().Debug("Error connecting to host %s", err.Error())
			zap.S().Fatalf(" Invalid (Username/Password/IP), use 'single quotes' to pass password")
		}

		c, err = pmk.NewClient(ctx.Fqdn, executor, ctx.AllowInsecure, false)
		if err != nil {
			zap.S().Fatalf("Unable to load clients needed for the Cmd. Error: %s", err.Error())
		}

		// Validate the user credentials entered during config set and will loop back again if invalid
		if err := validateUserCredentials(ctx, c); err != nil {
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

	zap.S().Info("==========Uploading supportBundle to S3 bucket==========")
	err := supportBundle.SupportBundleUpload(ctx, c)
	if err != nil {
		zap.S().Infof("Failed to upload pf9ctl supportBundle to %s bucket!!", supportBundle.S3_BUCKET_NAME)
	} else {

		fmt.Printf(color.Green("✓ ")+"Succesfully uploaded pf9ctl supportBundle to %s bucket at %s location \n",
			supportBundle.S3_BUCKET_NAME, supportBundle.S3_Location)
	}

	zap.S().Debug("==========Finished running supportBundleupload==========")
}
