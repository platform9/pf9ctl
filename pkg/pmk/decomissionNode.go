package pmk

import (
	"fmt"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/platform9/pf9ctl/pkg/client"
	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/objects"
	"github.com/platform9/pf9ctl/pkg/qbert"
	"github.com/platform9/pf9ctl/pkg/util"
	"go.uber.org/zap"
)

func removePf9Installation(c client.Client) {
	cmd := fmt.Sprintf("rm -rf %s", util.EtcDir)
	c.Executor.RunCommandWait(cmd)
	cmd = fmt.Sprintf("rm -rf %s", util.OptDir)
	c.Executor.RunCommandWait(cmd)
	cmd = fmt.Sprintf("rm -rf $HOME/pf9")
	c.Executor.RunCommandWait(cmd)
}

func removeHostagent(c client.Client, hostOS string) {

	var services = []string{"pf9-hostagent", "pf9-nodeletd", "pf9-kubelet"}
	//stop hostagent
	for _, service := range services {
		cmd := fmt.Sprintf("sudo systemctl stop %s", service)
		_, err := c.Executor.RunWithStdout("bash", "-c", cmd)
		if err != nil {
			zap.S().Debugf("Could not execute command %v", err)
		}
	}
	//remove hostagent
	var err error
	if hostOS == "debian" {
		_, err = c.Executor.RunWithStdout("bash", "-c", "sudo apt-get purge pf9-hostagent -y")
	} else {
		_, err = c.Executor.RunWithStdout("bash", "-c", "sudo yum remove pf9-hostagent -y")
	}
	if err != nil {
		zap.S().Fatalf("Could not execute command %v", err)
	}
	for _, file := range util.Files {
		cmd := fmt.Sprintf("rm -rf %s", file)
		c.Executor.RunCommandWait(cmd)
	}
}

func DecommissionNode(cfg *objects.Config, nc *objects.NodeConfig, removePf9 bool) {
	//Doc decommission steps
	//detach-node from cluster
	//deauthorize-node from controle plane
	//stop pf9-hostagent
	//stop pf9-nodeletd
	//stop pf9-kublet
	//purge pf9-hostagent
	//clean up logs

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Color("red")

	var executor cmdexec.Executor
	var err error

	if executor, err = cmdexec.GetExecutor(cfg.Spec.ProxyURL, util.Node, nc); err != nil {
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
	hostOS, err := ValidatePlatform(c.Executor)
	if err != nil {
		zap.S().Fatalf("Error getting OS version")
	}
	var nodeIPs []string
	nodeIPs = append(nodeIPs, ip)
	hostID := c.Resmgr.GetHostId(auth.Token, nodeIPs)
	//check if hostagent is installed on host
	if hostOS == "debian" {
		_, err = c.Executor.RunWithStdout("bash", "-c", "dpkg -s pf9-hostagent")
	} else {
		_, err = c.Executor.RunWithStdout("bash", "-c", "yum list installed pf9-hostagent")
	}
	if err == nil {
		//check if node is connected to any cluster
		var nodeInfo qbert.Node
		var nodeConnectedToDU bool
		if len(hostID) != 0 {
			nodeConnectedToDU = true
			nodeInfo = c.Qbert.GetNodeInfo(auth.Token, auth.ProjectID, hostID[0])
		}

		if nodeInfo.ClusterName == "" {
			fmt.Println(color.Green("✓ ") + "Node is not connected to any cluster")
			if nodeConnectedToDU {
				fmt.Println("Deauthorizing node:", hostID[0])
				err = c.Qbert.DeauthoriseNode(hostID[0], auth.Token)
				if err != nil {
					zap.S().Fatalf("Failed to deauthorize node")
				} else {
					fmt.Println(color.Green("✓ ") + "Deauthorized node from UI")
				}
				s.Start()
				s.Suffix = " Removing pf9-hostagent (this might take a few minutes...)"
				//s.FinalMSG = color.Green("✓ ") + "pf9-hostagent removed successfully"
				removeHostagent(c, hostOS)
				s.Stop()
				fmt.Println(color.Green("✓ ") + "pf9-hostagent removed successfully")

			} else {
				//case where node is not connected to DU but hostagent is installed partially
				s.Start()
				s.Suffix = " Removing pf9-hostagent (this might take a few minutes...)"
				//s.FinalMSG = color.Green("✓ ") + "pf9-hostagent removed successfully"
				removeHostagent(c, hostOS)
				s.Stop()
				fmt.Println(color.Green("✓ ") + "pf9-hostagent removed successfully")
			}
			//remove pf9 dir
			if removePf9 {
				removePf9Installation(c)
			}
			//fmt.Println("Node decommissioning started....This may take a few minutes....Check the latest status in UI")
			s.Restart()
			s.Suffix = " Node decommissioning started....This may take a few minutes....Check the latest status in UI"
			//s.FinalMSG = color.Green("✓ ") + "Node decommission completed"
			time.Sleep(50 * time.Second)
			s.Stop()
			fmt.Println(color.Green("✓ ") + "Node decommission completed")
		} else {
			//detach node from cluster
			fmt.Printf(color.Green("✓ ")+"Node is connected to %s cluster\n", nodeInfo.ClusterName)
			//fmt.Println("Detaching node from cluster...")
			s.Start()
			s.Suffix = " Detaching node from cluster..."
			err = c.Qbert.DetachNode(nodeInfo.ClusterUuid, auth.ProjectID, auth.Token, hostID[0])
			if err != nil {
				s.Stop()
				zap.S().Fatalf("Failed to detach host from cluster")
			} else {
				s.Stop()
				fmt.Println(color.Green("✓ ") + "Detached node from cluster")
			}

			//deauthorize host from UI
			//fmt.Println("Deauthorizing node from UI...")
			s.Restart()
			s.Suffix = " Deauthorizing node from UI..."
			err = c.Qbert.DeauthoriseNode(hostID[0], auth.Token)
			if err != nil {
				s.Stop()
				zap.S().Fatalf("Failed to deauthorize node")
			} else {
				s.Stop()
				fmt.Println(color.Green("✓ ") + "Deauthorized node from UI")
			}
			//stop host agent and remove it
			s.Restart()
			s.Suffix = " Removing pf9-hostagent (this might take a few minutes...)"
			removeHostagent(c, hostOS)
			s.Stop()
			fmt.Println(color.Green("✓ ") + "pf9-hostagent removed successfully")
			//remove pf9 dir
			if removePf9 {
				removePf9Installation(c)
			}
			s.Restart()
			s.Suffix = " Node decommissioning started....This may take a few minutes....Check the latest status in UI"
			time.Sleep(50 * time.Second)
			s.Stop()
			fmt.Println(color.Green("✓ ") + "Node decommission completed")
		}
	} else {
		fmt.Println("Host is not connected to Platform9 Management Plane")
	}
}
