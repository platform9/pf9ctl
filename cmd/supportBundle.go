// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"fmt"
	"time"

	"github.com/platform9/pf9ctl/pkg/client"
	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/config"
	"github.com/platform9/pf9ctl/pkg/objects"
	"github.com/platform9/pf9ctl/pkg/supportBundle"
	"github.com/platform9/pf9ctl/pkg/util"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// supportBundleCmd represents the supportBundle command
var (
	supportBundleCmd = &cobra.Command{
		Use:   "bundle",
		Short: "Gathers the support bundle and uploads it to S3",
		Long:  `Gathers support bundle that includes logs for pf9 services and pf9ctl, uploads to S3 `,
		Run:   supportBundleUpload,
	}

	bundleConfig objects.NodeConfig
)

//This initialization is using create commands which is not in use for now.
func init() {
	supportBundleCmd.Flags().StringVarP(&bundleConfig.User, "user", "u", "", "ssh username for the nodes")
	supportBundleCmd.Flags().StringVarP(&bundleConfig.Password, "password", "p", "", "ssh password for the nodes (use 'single quotes' to pass password)")
	supportBundleCmd.Flags().StringVarP(&bundleConfig.SshKey, "ssh-key", "s", "", "ssh key file for connecting to the nodes")
	supportBundleCmd.Flags().StringSliceVarP(&bundleConfig.IPs, "ip", "i", []string{}, "IP address of host to be prepared")
	supportBundleCmd.Flags().StringVar(&bundleConfig.MFA, "mfa", "", "MFA token")
	supportBundleCmd.Flags().StringVarP(&bundleConfig.SudoPassword, "sudo-pass", "e", "", "sudo password for user on remote host")

	rootCmd.AddCommand(supportBundleCmd)
}

func supportBundleUpload(cmd *cobra.Command, args []string) {
	zap.S().Debug("==========Running supportBundleUpload==========")

	detachedMode := cmd.Flags().Changed("no-prompt")
	isRemote := cmdexec.CheckRemote(bundleConfig)

	if isRemote {
		if !config.ValidateNodeConfig(&bundleConfig, !detachedMode) {
			zap.S().Fatal("Invalid remote node config (Username/Password/IP), use 'single quotes' to pass password")
		}
	}

	cfg := &objects.Config{WaitPeriod: time.Duration(60), AllowInsecure: false, MfaToken: bundleConfig.MFA}
	var err error
	if detachedMode {
		err = config.LoadConfig(util.Pf9DBLoc, cfg, bundleConfig)
	} else {
		err = config.LoadConfigInteractive(util.Pf9DBLoc, cfg, bundleConfig)
	}
	if err != nil {
		zap.S().Fatalf("Unable to load the context: %s\n", err.Error())
	}
	fmt.Println(color.Green("✓ ") + "Loaded Config Successfully")

	var executor cmdexec.Executor
	if executor, err = cmdexec.GetExecutor(cfg.ProxyURL, bundleConfig); err != nil {
		zap.S().Fatalf("Unable to create executor: %s\n", err.Error())
	}

	var c client.Client
	if c, err = client.NewClient(cfg.Fqdn, executor, cfg.AllowInsecure, false); err != nil {
		zap.S().Fatalf("Unable to create client: %s\n", err.Error())
	}

	defer c.Segment.Close()

	if isRemote {
		if err := SudoPasswordCheck(executor, detachedMode, bundleConfig.SudoPassword); err != nil {
			zap.S().Fatal("Failed executing commands on remote machine with sudo: ", err.Error())
		}
	}

	zap.S().Info("==========Uploading supportBundle to S3 bucket==========")
	err = supportBundle.SupportBundleUpload(*cfg, c, isRemote)
	if err != nil {
		zap.S().Infof("Failed to upload pf9ctl supportBundle to %s bucket!!", supportBundle.S3_BUCKET_NAME)
	} else {

		fmt.Printf(color.Green("✓ ")+"Succesfully uploaded pf9ctl supportBundle to %s bucket at %s location \n",
			supportBundle.S3_BUCKET_NAME, supportBundle.S3_Location)
	}

	zap.S().Debug("==========Finished running supportBundleupload==========")
}
