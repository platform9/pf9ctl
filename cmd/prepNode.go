// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/platform9/pf9ctl/pkg/client"
	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/config"
	"github.com/platform9/pf9ctl/pkg/log"
	"github.com/platform9/pf9ctl/pkg/objects"
	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/platform9/pf9ctl/pkg/ssh"
	"github.com/platform9/pf9ctl/pkg/supportBundle"
	"github.com/platform9/pf9ctl/pkg/util"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh/terminal"
)

// prepNodeCmd represents the prepNode command
var prepNodeCmd = &cobra.Command{
	Use:   "prep-node",
	Short: "Sets up prerequisites & prepares a node to use with PMK",
	Long: `Prepare a node to be ready to be added to a Kubernetes cluster. Read more
	at http://pf9.io/cli_clprep.`,
	Run: prepNodeRun,
	Args: func(prepNodeCmd *cobra.Command, args []string) error {
		if prepNodeCmd.Flags().Changed("disable-swapoff") {
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

var nodeConfig objects.NodeConfig

func init() {
	prepNodeCmd.Flags().StringVarP(&nodeConfig.User, "user", "u", "", "ssh username for the nodes")
	prepNodeCmd.Flags().StringVarP(&nodeConfig.Password, "password", "p", "", "ssh password for the nodes (use 'single quotes' to pass password)")
	prepNodeCmd.Flags().StringVarP(&nodeConfig.SshKey, "ssh-key", "s", "", "ssh key file for connecting to the nodes")
	prepNodeCmd.Flags().StringSliceVarP(&nodeConfig.IPs, "ip", "i", []string{}, "IP address of host to be prepared")
	prepNodeCmd.Flags().BoolVarP(&skipChecks, "skip-checks", "c", false, "Will skip optional checks if true")
	prepNodeCmd.Flags().BoolVarP(&disableSwapOff, "disable-swapoff", "d", false, "Will skip swapoff")
	prepNodeCmd.Flags().MarkHidden("disable-swapoff")
	prepNodeCmd.Flags().StringVar(&nodeConfig.MFA, "mfa", "", "MFA token")
	prepNodeCmd.Flags().StringVarP(&nodeConfig.SudoPassword, "sudo-pass", "e", "", "sudo password for user on remote host")
	prepNodeCmd.Flags().BoolVarP(&nodeConfig.RemoveExistingPkgs, "remove-existing-pkgs", "r", false, "Will remove previous installation if found (default false)")
	prepNodeCmd.Flags().BoolVar(&util.SkipKube, "skip-kube", false, "Skip installing pf9-kube/nodelet on this host")
	prepNodeCmd.Flags().MarkHidden("skip-kube")
	// At the moment prep-node command only install the kube role. If this changes in future, this option can be changed to something more generic.
	prepNodeCmd.Flags().StringVar(&util.KubeVersion, "kube-version", "", "Specific version of pf9-kube to install")
	prepNodeCmd.Flags().MarkHidden("kube-version")
	prepNodeCmd.Flags().BoolVar(&util.CheckIfOnboarded, "skip-connected", false, "If the node is already connected to the PMK control plane, prep-node will be skipped")

	rootCmd.AddCommand(prepNodeCmd)
}

func prepNodeRun(cmd *cobra.Command, args []string) {
	zap.S().Debug("==========Running prep-node==========")

	if skipChecks {
		pmk.WarningOptionalChecks = true
	}

	detachedMode := cmd.Flags().Changed("no-prompt")
	isRemote := cmdexec.CheckRemote(nodeConfig)

	if isRemote {
		if !config.ValidateNodeConfig(&nodeConfig, !detachedMode) {
			zap.S().Fatal("Invalid remote node config (Username/Password/IP), use 'single quotes' to pass password")
		}
	}

	cfg := &objects.Config{WaitPeriod: time.Duration(60), AllowInsecure: false, MfaToken: nodeConfig.MFA}
	var err error
	if detachedMode {
		nodeConfig.RemoveExistingPkgs = true
		err = config.LoadConfig(util.Pf9DBLoc, cfg, nodeConfig)
	} else {
		err = config.LoadConfigInteractive(util.Pf9DBLoc, cfg, nodeConfig)
	}

	if err != nil {
		zap.S().Fatalf("Unable to load the context: %s\n", err.Error())
	}

	fmt.Println(color.Green("✓ ") + "Loaded Config Successfully")
	zap.S().Debug("Loaded Config Successfully")
	var executor cmdexec.Executor
	if executor, err = cmdexec.GetExecutor(cfg.ProxyURL, nodeConfig); err != nil {
		zap.S().Fatalf("Unable to create executor: %s\n", err.Error())
	}

	var c client.Client
	if c, err = client.NewClient(cfg.Fqdn, executor, cfg.AllowInsecure, false); err != nil {
		zap.S().Fatalf("Unable to create client: %s\n", err.Error())
	}
	defer c.Segment.Close()
	// Fetch the keystone token.
	auth, err := c.Keystone.GetAuth(
		cfg.Username,
		cfg.Password,
		cfg.Tenant,
		cfg.MfaToken,
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
		if err := SudoPasswordCheck(executor, detachedMode, nodeConfig.SudoPassword); err != nil {
			zap.S().Fatal("Failed executing commands on remote machine with sudo: ", err.Error())
		}
	}

	// If all pre-requisite checks passed in Check-Node then prep-node
	result, err := pmk.CheckNode(*cfg, c, auth, nodeConfig)
	if err != nil {
		// Uploads pf9cli log bundle if pre-requisite checks fails
		errbundle := supportBundle.SupportBundleUpload(*cfg, c, isRemote)
		if errbundle != nil {
			zap.S().Debugf("Unable to upload supportbundle to s3 bucket %s", errbundle.Error())
		}
		zap.S().Fatalf("\nPre-requisite check(s) failed %s\n", err.Error())
	}

	if result == pmk.RequiredFail {
		zap.S().Fatalf(color.Red("x ")+"Required pre-requisite check(s) failed. See %s or use --verbose for logs \n", log.GetLogLocation(util.Pf9Log))
	} else if result == pmk.CleanInstallFail {
		fmt.Println("\nPrevious Installation Removed")
	}

	if result == pmk.OptionalFail {
		if !skipChecks {
			if detachedMode {
				fmt.Print(color.Red("x ") + "Optional pre-requisite check(s) failed. Use --skip-checks to skip these checks.\n")
				os.Exit(1)
			} else {
				fmt.Print("\nOptional pre-requisite check(s) failed. Do you want to continue? (y/n) ")
				reader := bufio.NewReader(os.Stdin)
				char, _, _ := reader.ReadRune()
				if char != 'y' {
					os.Exit(0)
				}
			}
		} else {
			fmt.Print("\nProceeding for prep-node with failed optional check(s)\n")
		}
	}

	if err := pmk.PrepNode(*cfg, c, auth); err != nil {

		// Uploads pf9cli log bundle if prepnode failed to get prepared
		errbundle := supportBundle.SupportBundleUpload(*cfg, c, isRemote)
		if errbundle != nil {
			zap.S().Debugf("Unable to upload supportbundle to s3 bucket %s", errbundle.Error())
		}

		zap.S().Debugf("Unable to prep node: %s\n", err.Error())
		zap.S().Fatalf("\nFailed to prepare node, error: %s. See %s or use --verbose for logs\n", err.Error(), log.GetLogLocation(util.Pf9Log))
	}

	zap.S().Debug("==========Finished running prep-node==========")
}

// To check if Remote Host needs Password to access Sudo and prompt for Sudo Password if exists.
func SudoPasswordCheck(exec cmdexec.Executor, detached bool, sudoPass string) error {

	ssh.SudoPassword = sudoPass

	_, err := exec.RunWithStdout("-l | grep '(ALL) PASSWD: ALL'")
	if err == nil {
		if detached {
			if sudoPass == "" {
				return errors.New("sudo password is required for the user on remote host, use --sudo-pass(-e) flag to pass")
			} else if validateSudoPassword(exec) == util.Invalid {
				return errors.New("Invalid password for user on remote host")
			}
		}

		// To bail out if Sudo Password entered is invalid multiple times.
		loopcounter := 1
		for true {
			if loopcounter >= 4 {
				zap.S().Fatalf("\n" + color.Red("x ") + "Invalid Sudo Password entered multiple times")
			}
			// Validate Sudo Password entered.
			if ssh.SudoPassword == "" || validateSudoPassword(exec) == util.Invalid {
				loopcounter += 1
				fmt.Printf("\n" + color.Red("x ") + "Invalid Sudo Password provided of Remote Host\n")
				fmt.Printf("Enter Sudo password for Remote Host: ")
				sudopassword, _ := terminal.ReadPassword(0)
				ssh.SudoPassword = string(sudopassword)
			} else {
				return nil
			}
		}
	}
	return nil
}

func validateSudoPassword(exec cmdexec.Executor) string {

	_ = pmk.CheckSudo(exec)
	// Validate Sudo Password entered for Remote Host from stderr.
	if strings.Contains(cmdexec.StdErrSudoPassword, util.InvalidPassword) {
		return util.Invalid
	}
	return util.Valid
}
