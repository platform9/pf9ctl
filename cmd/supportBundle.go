// Copyright © 2020 The pf9ctl authors

package cmd

import (
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

		executor, err := getExecutor()
		if err != nil {
			zap.S().Debug("Error connecting to host %s", err.Error())
			zap.S().Fatalf(" Invalid (Username/Password/IP)")
		}

		c, err = pmk.NewClient(ctx.Fqdn, executor, ctx.AllowInsecure, false)
		if err != nil {
			zap.S().Fatalf("Unable to load clients needed for the Cmd. Error: %s", err.Error())
		}

		// Validate the user credentials entered during config set and will loop back again if invalid
		if err := validateUserCredentials(ctx, c); err != nil {

			//Check if no or invalid config exists, then bail out if asked for correct config for maxLoop times.
			err = configValidation(pmk.LoopCounter)
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

	zap.S().Infof("==========Uploading pf9ctl log bundle to S3 bucket==========")
	err := supportBundle.SupportBundleUpload(ctx, c)
	if err != nil {
		zap.S().Fatalf("Unable to upload supportbundle to S3 bucket %s", err.Error())
	}

	zap.S().Debug("==========Finished running supportbundleupload==========")
}
