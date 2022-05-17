// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"fmt"
	"strings"

	"github.com/platform9/pf9ctl/pkg/client"
	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/config"
	"github.com/platform9/pf9ctl/pkg/log"
	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/platform9/pf9ctl/pkg/supportBundle"
	"github.com/platform9/pf9ctl/pkg/util"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	//nc *objects.NodeConfig

	checkNodeCmd = &cobra.Command{
		Use:   "check",
		Short: "Checks prerequisites on a node to use with PMK",
		Long: `Check if a node satisfies prerequisites to be ready to be added to a Kubernetes cluster. Read more
	at https://platform9.com/blog/support/managed-container-cloud-requirements-checklist/`,
		Run: checkNodeRun,
		PreRun: func(cmd *cobra.Command, args []string) {
			if node.Hostname != "" {
				nc.Spec.Nodes = append(nc.Spec.Nodes, node)
			}
		},
	}
)

func init() {
	// nc := objects.NodeConfig{}
	checkNodeCmd.Flags().StringVarP(&node.Hostname, "user", "u", "", "ssh username for the nodes")
	checkNodeCmd.Flags().StringVarP(&nc.Password, "password", "p", "", "ssh password for the nodes (use 'single quotes' to pass password)")
	checkNodeCmd.Flags().StringVarP(&nc.SshKey, "ssh-key", "s", "", "ssh key file for connecting to the nodes")
	checkNodeCmd.Flags().StringVarP(&node.Ip, "ip", "i", "", "IP address of host to be prepared")
	checkNodeCmd.Flags().StringVar(&util.MFA, "mfa", "", "MFA token")
	checkNodeCmd.Flags().StringVarP(&util.SudoPassword, "sudo-pass", "e", "", "sudo password for user on remote host")
	checkNodeCmd.Flags().BoolVarP(&util.RemoveExistingPkgs, "remove-existing-pkgs", "r", false, "Will remove previous installation if found (default false)")
	checkNodeCmd.Flags().StringVar(&ConfigPath, "user-config", "", "Path of user-config file")
	checkNodeCmd.Flags().StringVar(&NodeConfigPath, "node-config", "", "Path of node-config file")

	//checkNodeCmd.Flags().BoolVarP(&floatingIP, "floating-ip", "f", false, "") //Unsupported in first version.
	nodeCmd.AddCommand(checkNodeCmd)

}

func checkNodeRun(cmd *cobra.Command, args []string) {
	zap.S().Debug("==========Running check-node==========")

	if cmd.Flags().Changed("user-config") {
		util.Pf9DBLoc = ConfigPath
	}

	if cmd.Flags().Changed("node-config") {
		config.LoadNodeConfig(nc, NodeConfigPath)
	}

	detachedMode := cmd.Flags().Changed("no-prompt")
	isRemote := cmdexec.CheckRemote(nc)

	if isRemote {
		if !config.ValidateNodeConfig(nc, !detachedMode) {
			zap.S().Fatal("Invalid remote node config (Username/Password/IP), use 'single quotes' to pass password")
		}
	}

	/*cfg := &objects.Config{
		Spec: objects.UserData{
			MfaToken: nc.MFA,
			OtherData: objects.Other{
				WaitPeriod:    time.Duration(60),
				AllowInsecure: false,
			},
		},
	}*/
	var err error
	if detachedMode {
		util.RemoveExistingPkgs = true
		err = config.LoadConfig(util.Pf9DBLoc, cfg, nc)
	} else {
		err = config.LoadConfigInteractive(util.Pf9DBLoc, cfg, nc)
	}
	if err != nil {
		zap.S().Fatalf("Unable to load the context: %s\n", err.Error())
	}

	fmt.Println(color.Green("✓ ") + "Loaded Config Successfully")
	zap.S().Debug("Loaded Config Successfully")
	var executor cmdexec.Executor
	if executor, err = cmdexec.GetExecutor(cfg.Spec.ProxyURL, nc); err != nil {
		zap.S().Fatalf("Unable to create executor: %s\n", err.Error())
	}

	var c client.Client
	if c, err = client.NewClient(cfg.Spec.AccountUrl, executor, cfg.Spec.OtherData.AllowInsecure, false); err != nil {
		zap.S().Fatalf("Unable to create client: %s\n", err.Error())
	}

	defer c.Segment.Close()

	// Fetch the keystone token.
	auth, err := c.Keystone.GetAuth(
		cfg.Spec.Username,
		cfg.Spec.Password,
		cfg.Spec.Tenant,
		cfg.Spec.MfaToken,
	)

	if err != nil {
		// Certificate expiration is detected by the http library and
		// only error object gets populated, which means that the http
		// status code does not reflect the actual error code.
		// So parsing the err to check for certificate expiration.
		if strings.Contains(strings.ToLower(err.Error()), util.CertsExpireErr) {

			zap.S().Fatalf("Possible clock skew detected. Check the system time and retry.")
		}
		zap.S().Fatalf("Unable to obtain keystone credentials: %s", err.Error())
	}

	if isRemote {
		if err := SudoPasswordCheck(executor, detachedMode, util.SudoPassword); err != nil {
			zap.S().Fatal("Failed executing commands on remote machine with sudo: ", err.Error())
		}
	}

	result, err := pmk.CheckNode(*cfg, c, auth, nc)
	if err != nil {
		// Uploads pf9cli log bundle if checknode fails
		errbundle := supportBundle.SupportBundleUpload(*cfg, c, isRemote)
		if errbundle != nil {
			zap.S().Debugf("Unable to upload supportbundle to s3 bucket %s", errbundle.Error())
		}
		zap.S().Fatalf("Unable to perform pre-requisite checks on this node: %s", err.Error())
	}

	if result == pmk.RequiredFail {
		zap.S().Fatalf(color.Red("x ")+"Required pre-requisite check(s) failed. See %s or use --verbose for logs \n", log.GetLogLocation(util.Pf9Log))
		//this is so the exit flag is set to 1
	} else if result == pmk.OptionalFail {
		fmt.Printf("\nOptional pre-requisite check(s) failed. See %s or use --verbose for logs \n", log.GetLogLocation(util.Pf9Log))
	} else if result == pmk.CleanInstallFail {
		fmt.Println("\nPrevious Installation Removed")
	}
	zap.S().Debug("==========Finished running check-node==========")
}
