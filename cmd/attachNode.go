package cmd

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/platform9/pf9ctl/pkg/client"
	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/config"
	"github.com/platform9/pf9ctl/pkg/objects"
	"github.com/platform9/pf9ctl/pkg/util"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	masterIPs   []string
	workerIPs   []string
	clusterName string
	Errhostid   error
)

var (
	attachNodeCmd = &cobra.Command{
		Use:   "attach-node [flags] cluster-name",
		Short: "Attaches a node to the Kubernetes cluster",
		Long:  "Attach nodes to existing cluster. At a time, multiple workers but only one master can be attached",
		Args: func(attachNodeCmd *cobra.Command, args []string) error {
			// even if '-u' option is specified use some dummy cluster name
			if len(args) > 1 {
				return errors.New("only cluster name is accepted as a parameter")
			} else if len(args) < 1 {
				if clusterUuid == "" {
					return errors.New("either cluster name or cluster uuid is required for attach-node")
				} else 	{
					return nil
				}
			} else if clusterUuid != "" {
				return errors.New("only one of 'cluster name' or 'cluster uuid' can be specified in a single usage")
			}
			clusterName = args[0]
			return nil
		},
		Run: attachNodeRun,
	}

	attachconfig objects.NodeConfig
)

func init() {
	attachNodeCmd.Flags().StringSliceVarP(&masterIPs, "master-ip", "m", []string{}, "master node ip address")
	attachNodeCmd.Flags().StringSliceVarP(&workerIPs, "worker-ip", "w", []string{}, "worker node ip address")
	attachNodeCmd.Flags().StringVarP(&clusterUuid, "uuid", "u", "", "uuid of the cluster to attach the node to")
	attachNodeCmd.Flags().StringVar(&attachconfig.MFA, "mfa", "", "MFA token")
	rootCmd.AddCommand(attachNodeCmd)
	// '-u' option for inetrnal use only for now
	attachNodeCmd.Flags().MarkHidden("uuid")
}

func attachNodeRun(cmd *cobra.Command, args []string) {
	zap.S().Debug("==========Running Attach Node==========")

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

	defer c.Segment.Close()

	if len(masterIPs) == 0 && len(workerIPs) == 0 {
		zap.S().Fatalf("No nodes were specified to be attached to the cluster")
	}

	auth, err := c.Keystone.GetAuth(cfg.Username, cfg.Password, cfg.Tenant, cfg.MfaToken)
	if err != nil {
		zap.S().Debug("Failed to get keystone %s", err.Error())
	}
	projectId := auth.ProjectID
	token := auth.Token
	if clusterUuid != "" {
		if clusterName, err = c.Qbert.CheckClusterExistsWithUuid(clusterUuid, projectId, token); err != nil {
			zap.S().Fatalf("unable to verify cluster using uuid %s", err.Error())
		} else if clusterName == "" {
			zap.S().Fatalf("cluster with given uuid does not exist")
		}
	} else {
		_, clusterUuid, _ = c.Qbert.CheckClusterExists(clusterName, projectId, token)
	}

	clusterStatus := fetchClusterStatus(c.Executor, cfg.Fqdn, token, projectId, clusterUuid)

	if clusterStatus == "ok" {

		// master ips
		var masterHostIDs []string
		if len(masterIPs) > 0 {
			masterHostIDs = hostId(c.Executor, cfg.Fqdn, token, masterIPs)
		}

		// worker ips
		var workerHostIDs []string
		if len(workerIPs) > 0 {
			workerHostIDs = hostId(c.Executor, cfg.Fqdn, token, workerIPs)
		}

		// Attaching worker node(s) to cluster
		if err := c.Segment.SendEvent("Starting Attach-node", auth, "", ""); err != nil {
			zap.S().Debugf("Unable to send Segment event for attach node. Error: %s", err.Error())
		}
		if len(workerHostIDs) > 0 {
			fmt.Printf("Attaching node to the cluster %s\n", clusterName)
			var wokerids []string
			for _, worker := range workerHostIDs {
				if cname := isConnectedToAnyCluster(c.Executor, cfg.Fqdn, token, projectId, worker); cname != "null" {
					zap.S().Infof("Node with host id %s is connected to %s cluster", worker, cname)
				} else {
					wokerids = append(wokerids, worker)
				}
			}
			if len(wokerids) > 0 {
				err1 := c.Qbert.AttachNode(clusterUuid, projectId, token, wokerids, "worker")

				if err1 != nil {
					if err := c.Segment.SendEvent("Attaching-node", auth, "Failed to attach worker node", ""); err != nil {
						zap.S().Debugf("Unable to send Segment event for attach node. Error: %s", err.Error())
					}
					zap.S().Info("Encountered an error while attaching worker node to a Kubernetes cluster : ", err1)
				} else {
					if err := c.Segment.SendEvent("Attaching-node", auth, "Worker node attached", ""); err != nil {
						zap.S().Debugf("Unable to send Segment event for attach node. Error: %s", err.Error())
					}
					zap.S().Infof("Worker node(s) %v attached to cluster", wokerids)
				}
			} else {
				zap.S().Infof("No worker node available to attach to the cluster")
			}

		}
		// Attaching master node(s) to cluster
		if len(masterHostIDs) > 0 {
			fmt.Printf("Attaching node to the cluster %s\n", clusterName)
			var masterids []string
			for _, master := range masterHostIDs {
				if cname := isConnectedToAnyCluster(c.Executor, cfg.Fqdn, token, projectId, master); cname != "null" {
					zap.S().Infof("Node with host id %s is connected to %s cluster", master, cname)
				} else {
					masterids = append(masterids, master)
				}
			}
			if len(masterids) > 0 {
				err1 := c.Qbert.AttachNode(clusterUuid, projectId, token, masterids, "master")

				if err1 != nil {
					if err := c.Segment.SendEvent("Attaching-node", auth, "Failed to attach master node", ""); err != nil {
						zap.S().Debugf("Unable to send Segment event for attach node. Error: %s", err.Error())
					}
					zap.S().Info("Encountered an error while attaching master node to a Kubernetes cluster : ", err1)
				} else {
					if err := c.Segment.SendEvent("Attaching-node", auth, "Master node attached", ""); err != nil {
						zap.S().Debugf("Unable to send Segment event for attach node. Error: %s", err.Error())
					}
					zap.S().Infof("Master node(s) %v attached to cluster", masterids)
				}
			} else {
				zap.S().Infof("No master node available to attach to the cluster")
			}

		}
	} else {
		zap.S().Fatalf("Cluster is not ready. cluster status is %v", clusterStatus)
	}

}

func hostId(exec cmdexec.Executor, fqdn string, token string, IPs []string) []string {
	zap.S().Debug("Getting host IDs")
	var hostIdsList []string
	tkn := fmt.Sprintf(`"X-Auth-Token: %v"`, token)
	for _, ip := range IPs {
		ip = strings.TrimSpace(ip)
		ip = fmt.Sprintf(`"%v"`, ip)
		cmd := fmt.Sprintf("curl -sH %v -X GET %v/resmgr/v1/hosts | jq -r '.[] | select(.extensions!=\"\")  | select(.extensions.ip_address.data[]==(%v)) | .id' ", tkn, fqdn, ip)
		output, err := exec.RunWithStdout("bash", "-c", cmd)
		if err != nil {
			zap.S().Debug("Failed to get host ID for IP '%s': %v", ip, err)
		}
		hostID := strings.TrimSpace(output)
		if len(hostID) == 0 {
			zap.S().Infof("Unable to find host with IP %v please try again or run prep-node first", ip)
		} else {
			hostIdsList = append(hostIdsList, hostID)
		}
	}
	return hostIdsList
}

func fetchClusterStatus(exec cmdexec.Executor, fqdn string, token string, projectID string, clusterID string) string {
	zap.S().Debug("Getting cluster status")
	tkn := fmt.Sprintf(`"X-Auth-Token: %v"`, token)
	cmd := fmt.Sprintf("curl -sH %v -X GET %v/qbert/v3/%v/clusters/%v | jq '.status' ", tkn, fqdn, projectID, clusterID)
	status, err := exec.RunWithStdout("bash", "-c", cmd)
	if err != nil {
		zap.S().Fatalf("Unable to get cluster status : ", err)
	}
	status = strings.TrimSpace(strings.Trim(status, "\n\""))
	zap.S().Debug("Cluster status is : ", status)
	return status
}

// Check if the node being attached is already attached to any cluster
func isConnectedToAnyCluster(exec cmdexec.Executor, fqdn string, token string, projectID string, hostId string) string {
	zap.S().Debug("Checking if node is connected to any cluster")
	tkn := fmt.Sprintf(`"X-Auth-Token: %v"`, token)
	cmd := fmt.Sprintf("curl -sH %v -X GET %v/qbert/v3/%v/nodes/%v | jq '.clusterName' ", tkn, fqdn, projectID, hostId)
	clusterName, err := exec.RunWithStdout("bash", "-c", cmd)
	if err != nil {
		zap.S().Debug("Unable to check if node is connected to any cluster")
	}
	clusterName = strings.TrimSpace(strings.Trim(clusterName, "\n\""))
	return clusterName
}
