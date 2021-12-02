package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/platform9/pf9ctl/pkg/client"
	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/config"
	"github.com/platform9/pf9ctl/pkg/objects"
	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/platform9/pf9ctl/pkg/util"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var decommissionNodeCmd = &cobra.Command{
	Use:   "decommission-node",
	Short: "Decomisisons this node from the Platform9 control plane",
	Long:  "Removes the host agent package and decommissions this node from the Platform9 control plane.",
	Args: func(deauthNodeCmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errors.New("No parameters are needed")
		}
		return nil
	},
	Run: decommissionNodeRun,
}

func init() {
	decommissionNodeCmd.Flags().StringVar(&attachconfig.MFA, "mfa", "", "MFA token")
	rootCmd.AddCommand(decommissionNodeCmd)
}

func RunCommandWait(command string) {
	output := exec.Command("/bin/sh", "-c", command)
	output.Stdout = os.Stdout
	output.Stdin = os.Stdin
	err := output.Start()
	output.Wait()
	if err != nil {
		fmt.Println(err.Error())
	}
}

func decommissionNodeRun(cmd *cobra.Command, args []string) {

	detachedMode := cmd.Flags().Changed("no-prompt")

	if cmdexec.CheckRemote(nc) {
		if !config.ValidateNodeConfig(&nc, !detachedMode) {
			zap.S().Fatal("Invalid remote node config (Username/Password/IP), use 'single quotes' to pass password")
		}
	}

	cfg := &objects.Config{WaitPeriod: time.Duration(60), AllowInsecure: false, MfaToken: attachconfig.MFA}
	var err error
	if detachedMode {
		err = config.LoadConfig(util.Pf9DBLoc, cfg, nc)
	} else {
		err = config.LoadConfigInteractive(util.Pf9DBLoc, cfg, nc)
	}
	if err != nil {
		zap.S().Fatalf("Unable to load the context: %s\n", err.Error())
	}
	fmt.Println(color.Green("✓ ") + "Loaded Config Successfully")

	var executor cmdexec.Executor
	if executor, err = cmdexec.GetExecutor(cfg.ProxyURL, nc); err != nil {
		zap.S().Fatalf("Unable to create executor: %s\n", err.Error())
	}

	var c client.Client
	if c, err = client.NewClient(cfg.Fqdn, executor, cfg.AllowInsecure, false); err != nil {
		zap.S().Fatalf("Unable to create client: %s\n", err.Error())
	}

	auth, err := c.Keystone.GetAuth(cfg.Username, cfg.Password, cfg.Tenant, cfg.MfaToken)
	if err != nil {
		zap.S().Debug("Failed to get keystone %s", err.Error())
	}

	var nodeIPs []string
	nodeIPs = append(nodeIPs, getIp().String())
	token := auth.Token
	nodeUuid, _ := hostId(c.Executor, cfg.Fqdn, token, nodeIPs)

	if len(nodeUuid) == 0 {
		zap.S().Fatalf("Could not remove the node from the UI, check if the host agent is installed.")
	}

	version, _ := pmk.OpenOSReleaseFile(executor)

	if err != nil {
		zap.S().Fatalf("Error getting OS version")
	}

	if strings.Contains(string(version), util.Ubuntu) {
		fmt.Println("Removing packages")
		RunCommandWait("sudo dpkg --remove pf9-comms pf9-kube pf9-hostagent pf9-muster")
		fmt.Println("Purging packages")
		RunCommandWait("sudo dpkg --purge pf9-comms pf9-kube pf9-hostagent pf9-muster")
		fmt.Println("Removing /etc/pf9 logs")
		RunCommandWait("sudo rm -rf /etc/pf9")
		fmt.Println("Removing /opt/pf9 logs")
		RunCommandWait("sudo rm -rf /opt/pf9")
		fmt.Println("Removing pf9 HOME dir")
		RunCommandWait("sudo rm -rf $HOME/pf9")

		RunCommandWait("sudo pkill -9 `pidof kubelet`")
		RunCommandWait("sudo pkill -9 `pidof etcd`")
		RunCommandWait("sudo pkill -9 `pidof kube_proxy`")

		RunCommandWait("sudo rm -rf /opt/cni")
		RunCommandWait("sudo rm -rf /opt/containerd")
		RunCommandWait("sudo rm -rf /var/lib/containerd")

	} else {
		//command = "sudo yum erase -y pf9-hostagent -y"
		fmt.Println("Removing packages")
		RunCommandWait("sudo yum erase -y pf9-comms")
		RunCommandWait("sudo yum erase -y pf9-kube")
		RunCommandWait("sudo yum erase -y pf9-hostagent")
		RunCommandWait("sudo yum erase -y pf9-muster")
		fmt.Println("Removing /etc/pf9 logs")
		RunCommandWait("sudo rm -rf /etc/pf9")
		fmt.Println("Removing /opt/pf9 logs")
		RunCommandWait("sudo rm -rf /opt/pf9")
		fmt.Println("Removing pf9 HOME dir")
		RunCommandWait("sudo rm -rf $HOME/pf9")

		RunCommandWait("sudo pkill -9 `pidof kubelet`")
		RunCommandWait("sudo pkill -9 `pidof etcd`")
		RunCommandWait("sudo pkill -9 `pidof kube_proxy`")

		RunCommandWait("sudo rm -rf /opt/cni")
		RunCommandWait("sudo rm -rf /opt/containerd")
		RunCommandWait("sudo rm -rf /var/lib/containerd")

	}

	err = c.Qbert.DeauthoriseNode(nodeUuid[0], token)

	if err != nil {
		zap.S().Fatalf("Error removing the node from the UI ", err.Error())
	}
	fmt.Println("Removed the node from the UI")

	fmt.Println("Node decommissioning started....This may take a few minutes....Check the latest status in UI")

}
