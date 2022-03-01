package pmk

import (
	"fmt"
	"strings"
	"time"

	"github.com/platform9/pf9ctl/pkg/client"
	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/objects"
	"github.com/platform9/pf9ctl/pkg/util"
	"go.uber.org/zap"
)

func removePf9Instation(c client.Client) {
	commands := map[string]string{
		"Removing /etc/pf9 logs": "rm -rf /etc/pf9",
		"Removing /opt/pf9 logs": "rm -rf /opt/pf9",
		"Removing pf9 HOME dir":  "rm -rf $HOME/pf9",
	}

	for msg, cmd := range commands {
		fmt.Println(msg)
		c.Executor.RunCommandWait(cmd)
	}
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
	ip, err := c.Executor.RunWithStdout("/bin/sh", "-c", "hostname -I")
	ip = strings.TrimSpace(ip)
	if err != nil {
		zap.S().Debugf("unable to get host ip")
	}
	var nodeIPs []string
	nodeIPs = append(nodeIPs, ip)
	token := auth.Token
	nodeUuid := c.Resmgr.GetHostId(token, nodeIPs)
	if len(nodeUuid) == 0 {
		zap.S().Fatalf("Could not remove the node from the UI, check if the host agent is installed.")
	}

	version, err := OpenOSReleaseFile(executor)

	if err != nil {
		zap.S().Fatalf("Error getting OS version")
	}

	fmt.Println("Removing packages...")
	if strings.Contains(string(version), util.Ubuntu) {
		c.Executor.RunCommandWait("dpkg --remove pf9-comms pf9-kube pf9-hostagent pf9-muster")
		fmt.Println("Purging packages")
		c.Executor.RunCommandWait("dpkg --purge pf9-comms pf9-kube pf9-hostagent pf9-muster")

	} else {

		commands := []string{
			"yum erase -y pf9-comms",
			"yum erase -y pf9-kube",
			"yum erase -y pf9-hostagent",
			"yum erase -y pf9-muster",
		}
		for _, cmd := range commands {
			c.Executor.RunCommandWait(cmd)
		}
	}

	if removePf9 {
		removePf9Instation(c)
	}

	commands := []string{
		"pkill kubelet",
		"pkill etcd",
		"pkill kube-proxy",
		"pkill kube-apiserve",
		"pkill kube-schedule",
		"pkill kube-controll",
		"rm -rf /opt/cni",
		"rm -rf /opt/containerd",
		"rm -rf /var/lib/containerd",
		"rm -rf /var/opt/pf9",
	}

	for _, cmd := range commands {
		c.Executor.RunCommandWait(cmd)
	}

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
