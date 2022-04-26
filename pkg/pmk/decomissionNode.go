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
	fmt.Println("Removing /etc/pf9 logs")
	cmd := fmt.Sprintf("rm -rf %s", util.EtcDir)
	c.Executor.RunCommandWait(cmd)
	fmt.Println("Removing /var/opt/pf9 logs")
	cmd = fmt.Sprintf("rm -rf %s", util.OptDir)
	c.Executor.RunCommandWait(cmd)
	fmt.Println("Removing pf9 HOME dir")
	cmd = fmt.Sprintf("rm -rf $HOME/pf9")
	c.Executor.RunCommandWait(cmd)
}

func DecommissionNode(cfg *objects.Config, nc objects.NodeConfig, removePf9 bool) {

	var executor cmdexec.Executor
	var err error
	if executor, err = cmdexec.GetExecutor(cfg.Spec.ProxyURL, nc); err != nil {
		zap.S().Fatalf("Unable to create executor: %s\n", err.Error())
	}
	var c client.Client
	if c, err = client.NewClient(cfg.Spec.AccountUrl, executor, cfg.Spec.OtherData.AllowInsecure, false); err != nil {
		zap.S().Fatalf("Unable to create client: %s\n", err.Error())
	}
	auth, err := c.Keystone.GetAuth(cfg.Spec.Username, cfg.Spec.Password, cfg.Spec.Tenant, cfg.Spec.MfaToken)
	if err != nil {
		zap.S().Debug("Failed to get keystone %s", err.Error())
	}

	ip, err := c.Executor.RunWithStdout("bash", "-c", "hostname -I")
	//Handling case where host can have multiple IPs
	ip = strings.Split(ip, " ")[0]
	ip = strings.TrimSpace(ip)
	if err != nil {
		zap.S().Fatalf("ERROR : unable to get host ip")
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
		out := c.Executor.RunCommandWait("dpkg --remove pf9-comms pf9-kube pf9-hostagent pf9-muster")
		fmt.Println(out)
		fmt.Println("Purging packages")
		out = c.Executor.RunCommandWait("dpkg --purge pf9-comms pf9-kube pf9-hostagent pf9-muster")
		fmt.Println(out)

	} else {
		for _, p := range util.Pf9Packages {
			cmd := fmt.Sprintf("yum erase -y %s", p)
			out := c.Executor.RunCommandWait(cmd)
			fmt.Println(out)
		}
	}

	if removePf9 {
		removePf9Instation(c)
	}

	for _, service := range util.ProcessesList {
		cmd := fmt.Sprintf("pkill %s", service)
		c.Executor.RunCommandWait(cmd)
	}

	for _, file := range util.Files {
		cmd := fmt.Sprintf("rm -rf %s", file)
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
