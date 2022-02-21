package pmk

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/platform9/pf9ctl/pkg/client"
	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/objects"
	"github.com/platform9/pf9ctl/pkg/util"
	"go.uber.org/zap"
)

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

func removePf9Instation() {
	fmt.Println("Removing /etc/pf9 logs")
	RunCommandWait("sudo rm -rf /etc/pf9")
	fmt.Println("Removing /opt/pf9 logs")
	RunCommandWait("sudo rm -rf /opt/pf9")
	fmt.Println("Removing pf9 HOME dir")
	RunCommandWait("sudo rm -rf $HOME/pf9")

}

func DecommissionNode(cfg *objects.Config, nc objects.NodeConfig, removePf9 bool) {

	var executor cmdexec.Executor
	var err error
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
	nodeIPs = append(nodeIPs, GetIp().String())
	token := auth.Token
	nodeUuid := HostId(c.Executor, cfg.Fqdn, token, nodeIPs)

	if len(nodeUuid) == 0 {
		zap.S().Fatalf("Could not remove the node from the UI, check if the host agent is installed.")
	}

	version, err := OpenOSReleaseFile(executor)

	if err != nil {
		zap.S().Fatalf("Error getting OS version")
	}

	fmt.Println("Removing packages...")
	if strings.Contains(string(version), util.Ubuntu) {
		RunCommandWait("sudo dpkg --remove pf9-comms pf9-kube pf9-hostagent pf9-muster")
		fmt.Println("Purging packages")
		RunCommandWait("sudo dpkg --purge pf9-comms pf9-kube pf9-hostagent pf9-muster")

	} else {

		RunCommandWait("sudo yum erase -y pf9-comms")
		RunCommandWait("sudo yum erase -y pf9-kube")
		RunCommandWait("sudo yum erase -y pf9-hostagent")
		RunCommandWait("sudo yum erase -y pf9-muster")
	}

	if removePf9 {
		removePf9Instation()
	}

	RunCommandWait("sudo pkill kubelet")
	RunCommandWait("sudo pkill etcd")
	RunCommandWait("sudo pkill kube-proxy")
	RunCommandWait("sudo pkill kube-apiserve")
	RunCommandWait("sudo pkill kube-schedule")
	RunCommandWait("sudo pkill kube-controll")

	RunCommandWait("sudo rm -rf /opt/cni")
	RunCommandWait("sudo rm -rf /opt/containerd")
	RunCommandWait("sudo rm -rf /var/lib/containerd")
	RunCommandWait("sudo rm -rf /var/opt/pf9")
	RunCommandWait("sudo rm -rf /var/log/pf9")

	err = c.Qbert.DeauthoriseNode(nodeUuid[0], token)

	if err != nil {
		zap.S().Fatalf("Error removing the node from the UI ", err.Error())
	}
	fmt.Println("Removed the node from the UI")

	fmt.Println("Node decommissioning started....This may take a few minutes....Check the latest status in UI")
	// Wating for ports to close
	// Some ports taking time to close even after killing the process
	time.Sleep(50 * time.Second)

}
