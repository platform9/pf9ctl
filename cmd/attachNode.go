package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/pmk"
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

var attachNodeCmd = &cobra.Command{
	Use:   "attach-node [flags] cluster-name",
	Short: "attaches node to kubernetes cluster",
	Long:  "Attach nodes to existing cluster. At a time, multiple workers but only one master can be attached",
	Args: func(attachNodeCmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return errors.New("Only cluster name is accepted as a parameter")
		} else if len(args) < 1 {
			return errors.New("Cluster name is required for attach-node")
		}
		clusterName = args[0]
		return nil
	},
	Run: attachNodeRun,
}

func init() {
	attachNodeCmd.Flags().StringSliceVarP(&masterIPs, "master-ip", "m", []string{}, "master node ip address")
	attachNodeCmd.Flags().StringSliceVarP(&workerIPs, "worker-ip", "w", []string{}, "worker node ip address")
	rootCmd.AddCommand(attachNodeCmd)
}

func attachNodeRun(cmd *cobra.Command, args []string) {
	zap.S().Debug("==========Running Attach Node==========")
	// This flag is used to loop back if user enters invalid credentials during config set.
	credentialFlag = true
	// To bail out if loop runs recursively more than thrice
	pmk.LoopCounter = 0

	for credentialFlag {

		ctx, err = pmk.LoadConfig(util.Pf9DBLoc)
		if err != nil {
			zap.S().Fatalf("Unable to load the context: %s\n", err.Error())
		}

		executor, err := getExecutor(ctx.ProxyURL)
		if err != nil {
			zap.S().Debug("Error connecting to host %s", err.Error())
			zap.S().Fatalf(" Invalid (Username/Password/IP)")
		}

		c, err = pmk.NewClient(ctx.Fqdn, executor, ctx.AllowInsecure, false)
		if err != nil {
			zap.S().Fatalf("Unable to load clients needed for the Cmd. Error: %s", err.Error())
		}

		// Validate the user credentials entered during config set and will loop back again if invalid
		if err := validateUserCredentials(ctx, c); err != nil {
			clearContext(&pmk.Context)
			//Check if no or invalid config exists, then bail out if asked for correct config for maxLoop times.
			err = configValidation(RegionInvalid, pmk.LoopCounter)
		} else {
			// We will store the set config if its set for first time using check-node
			if pmk.IsNewConfig {
				if err := pmk.StoreConfig(ctx, util.Pf9DBLoc); err != nil {
					zap.S().Errorf("Failed to store config: %s", err.Error())
				} else {
					pmk.IsNewConfig = false
				}
			}
			credentialFlag = false
		}
	}

	defer c.Segment.Close()

	if len(masterIPs) == 0 && len(workerIPs) == 0 {
		zap.S().Fatalf("No nodes were specified to be attached to the cluster")
	}

	auth, err := c.Keystone.GetAuth(ctx.Username, ctx.Password, ctx.Tenant)
	if err != nil {
		zap.S().Debug("Failed to get keystone %s", err.Error())
	}
	projectId := auth.ProjectID
	token := auth.Token

	//master ips
	var master_hostIds []string
	if len(masterIPs) > 0 {
		var err error
		master_hostIds, err = hostId(c.Executor, ctx.Fqdn, token, masterIPs)
		if err != nil {
			zap.S().Fatalf("%v", err)
		}
	}

	//worker ips
	var worker_hostIds []string
	if len(workerIPs) > 0 {
		var err error
		worker_hostIds, err = hostId(c.Executor, ctx.Fqdn, token, workerIPs)
		if err != nil {
			zap.S().Fatalf("%v", err)
		}
	}

	_, cluster_uuid, _ := c.Qbert.CheckClusterExists(clusterName, projectId, token)
	clusterStatus := cluster_Status(c.Executor, ctx.Fqdn, token, projectId, cluster_uuid)
	if clusterStatus == "ok" {
		//Attaching worker node(s) to cluster
		if err := c.Segment.SendEvent("Starting Attach-node", auth, "", ""); err != nil {
			zap.S().Errorf("Unable to send Segment event for attach node. Error: %s", err.Error())
		}
		if len(worker_hostIds) > 0 {
			err1 := c.Qbert.AttachNode(cluster_uuid, projectId, token, worker_hostIds, "worker")
			if err1 != nil {
				if err := c.Segment.SendEvent("Attaching-node", auth, "Failed to attach worker node", ""); err != nil {
					zap.S().Errorf("Unable to send Segment event for attach node. Error: %s", err.Error())
				}
				zap.S().Info("Encountered an error while attaching worker node to a Kubernetes cluster : ", err1)
			} else {
				if err := c.Segment.SendEvent("Attaching-node", auth, "Worker node attached", ""); err != nil {
					zap.S().Errorf("Unable to send Segment event for attach node. Error: %s", err.Error())
				}
				zap.S().Infof("Worker node(s) %v attached to cluster", worker_hostIds)
			}
		}
		//Attaching master node(s) to cluster
		if len(master_hostIds) > 0 {
			err1 := c.Qbert.AttachNode(cluster_uuid, projectId, token, master_hostIds, "master")
			if err1 != nil {
				if err := c.Segment.SendEvent("Attaching-node", auth, "Failed to attach master node", ""); err != nil {
					zap.S().Errorf("Unable to send Segment event for attach node. Error: %s", err.Error())
				}
				zap.S().Info("Encountered an error while attaching master node to a Kubernetes cluster : ", err1)
			} else {
				if err := c.Segment.SendEvent("Attaching-node", auth, "Master node attached", ""); err != nil {
					zap.S().Errorf("Unable to send Segment event for attach node. Error: %s", err.Error())
				}
				zap.S().Infof("Master node(s) %v attached to cluster", master_hostIds)
			}
		}
	} else {
		zap.S().Fatalf("Cluster is not ready. cluster status is %v", clusterStatus)
	}

}

func hostId(exec cmdexec.Executor, fqdn string, token string, IPs []string) ([]string, error) {
	zap.S().Debug("Getting host IDs")
	var hostIdsList []string
	tkn := fmt.Sprintf(`"X-Auth-Token: %v"`, token)
	for _, ip := range IPs {
		ip = fmt.Sprintf(`"%v"`, ip)
		cmd := fmt.Sprintf("curl -sH %v -X GET %v/resmgr/v1/hosts | jq -r '.[] | select(.info.responding==true) | select(.extensions.ip_address.data[]==(%v)) | .id' ", tkn, fqdn, ip)
		hostid, _ := exec.RunWithStdout("bash", "-c", cmd)
		hostid = strings.TrimSpace(strings.Trim(hostid, "\n"))
		if len(hostid) == 0 {
			Errhostid = fmt.Errorf("Unable to find host with IP %v please try again or run prep-node first", ip)
			return hostIdsList, Errhostid
		} else {
			hostIdsList = append(hostIdsList, hostid)
		}
	}
	return hostIdsList, nil
}

func cluster_Status(exec cmdexec.Executor, fqdn string, token string, projectID string, clusterID string) string {
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
