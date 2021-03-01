// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"fmt"

	"github.com/platform9/pf9ctl/pkg/keystone"
	"github.com/platform9/pf9ctl/pkg/log"
	"github.com/platform9/pf9ctl/pkg/pmk"
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

var (
	ctx  pmk.Config
	c    pmk.Client
	err  error
	auth keystone.KeystoneAuth
	// This flag helps us to loop-back the config set until the user enters valid credentials.
	credentialsFlag = true
)

func init() {
	checkNodeCmd.Flags().StringVarP(&user, "user", "u", "", "ssh username for the nodes")
	checkNodeCmd.Flags().StringVarP(&password, "password", "p", "", "ssh password for the nodes")
	checkNodeCmd.Flags().StringVarP(&sshKey, "ssh-key", "s", "", "ssh key file for connecting to the nodes")
	checkNodeCmd.Flags().StringSliceVarP(&ips, "ip", "i", []string{}, "IP address of host to be prepared")
	//checkNodeCmd.Flags().BoolVarP(&floatingIP, "floating-ip", "f", false, "") //Unsupported in first version.

	rootCmd.AddCommand(checkNodeCmd)
}

func checkNodeRun(cmd *cobra.Command, args []string) {
	zap.S().Debug("==========Running check-node==========")

	// Loading the config if exists or creating config and storing it if valid.
	for credentialsFlag {

		zap.S().Debug("Loading the config if exist or setting config and validating the user credentials")
		ctx, c, auth, credentialsFlag = configLoadAndValidate()
	}

	defer c.Segment.Close()

	result, err := pmk.CheckNode(ctx, c, auth)
	if err != nil {
		zap.S().Fatalf("Unable to perform pre-requisite checks on this node: %s", err.Error())
	}
	if result == pmk.RequiredFail {
		fmt.Printf("\nRequired pre-requisite check(s) failed. See %s or use --verbose for logs \n", log.GetLogLocation(util.Pf9Log))
	} else if result == pmk.OptionalFail {
		fmt.Printf("\nOptional pre-requisite check(s) failed. See %s or use --verbose for logs \n", log.GetLogLocation(util.Pf9Log))
	}

	zap.S().Debug("==========Finished running check-node==========")
}

func configLoadAndValidate() (pmk.Config, pmk.Client, keystone.KeystoneAuth, bool) {

	ctx, err = pmk.LoadConfig(util.Pf9DBLoc)
	if err != nil {
		zap.S().Fatalf("Unable to load the context: %s\n", err.Error())
	}

	executor, err := getExecutor()
	if err != nil {
		zap.S().Fatalf("Error connecting to host %s", err.Error())
	}

	c, err = pmk.NewClient(ctx.Fqdn, executor, ctx.AllowInsecure, false)
	if err != nil {
		zap.S().Fatalf("Unable to load clients needed for the Cmd. Error: %s", err.Error())
	}
	// Validating the config received after loading.
	auth, credentialsFlag = authenticateUserCredentials(c, ctx)

	return ctx, c, auth, credentialsFlag
}

func authenticateUserCredentials(pmk.Client, pmk.Config) (keystone.KeystoneAuth, bool) {

	zap.S().Debug("==========Validating the User Credentials==========")

	// Validating the credentials enterd (Username, Password, Service) by user using config setting.
	auth, err = c.Keystone.GetAuth(
		ctx.Username,
		ctx.Password,
		ctx.Tenant,
	)

	// If the credentials are invalid we will loop-back the config set before storing it till the credentials entered are valid.
	if err != nil {
		zap.S().Info("Invalid credentials entered (Username/Password/Tenant)\n")
	} else if pmk.IfNewConfig {
		if err := pmk.StoreConfig(ctx, util.Pf9DBLoc); err != nil {
			zap.S().Errorf("Failed to store config: %s", err.Error())
		} else {
			credentialsFlag = false
		}
	} else {
		credentialsFlag = false
	}

	return auth, credentialsFlag
}
