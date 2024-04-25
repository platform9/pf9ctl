package cmd

import (
	"errors"
	"fmt"
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
			if len(args) > 1 {
				return errors.New("only cluster name is accepted as a parameter")
			} else if len(args) < 1 {
				if clusterUuid == "" {
					return errors.New("either cluster name or cluster uuid is required for attach-node")
				} else {
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
	zap.S().Debug("Loaded Config Successfully")
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
		zap.S().Fatalf("Failed to get keystone %s", err.Error())
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
		_, clusterUuid, _, err = c.Qbert.CheckClusterExists(clusterName, projectId, token)
		if err != nil {
			zap.S().Fatalf("unable to fetch clusterUuid from cluster-name. Error: %s", err.Error())
		}
	}
	_, _, clusterStatus, err := c.Qbert.CheckClusterExists(clusterName, projectId, token)
	if err != nil {
		zap.S().Fatalf("unable to fetch cluster status from cluster-name. Error: %s", err.Error())
	}

	if clusterStatus == "ok" {

		// master ips
		var masterHostIDs []string
		if len(masterIPs) > 0 {
			masterHostIDs = c.Resmgr.GetHostId(token, masterIPs)
		}

		// worker ips
		var workerHostIDs []string
		if len(workerIPs) > 0 {
			workerHostIDs = c.Resmgr.GetHostId(token, workerIPs)
		}

		// Attaching worker node(s) to cluster
		if err := c.Segment.SendEvent("Starting Attach-node", auth, "", ""); err != nil {
			zap.S().Debugf("Unable to send Segment event for attach node. Error: %s", err.Error())
		}
		if len(workerHostIDs) > 0 {
			fmt.Printf("Attaching node to the cluster %s\n", clusterName)
			var wokerids []string
			for _, worker := range workerHostIDs {
				cname, err := c.Qbert.GetNodeInfo(token, projectId, worker)
				if err != nil {
					zap.S().Fatalf("Failed to get node info for host %s: %s", worker, err.Error())
				} else if cname.ClusterName != "" {
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
					zap.S().Fatal("Encountered an error while attaching worker node to a Kubernetes cluster : ", err1)
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
				cname, err := c.Qbert.GetNodeInfo(token, projectId, master)
				if err != nil {
					zap.S().Fatalf("Failed to get node info for host %s: %s", master, err.Error())
				} else if cname.ClusterName != "" {
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
					zap.S().Fatal("Encountered an error while attaching master node to a Kubernetes cluster : ", err1)
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
