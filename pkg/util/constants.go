package util

import (
	"os"
	"path/filepath"
	"time"
)

var Pf9Packages []string
var RequiredPorts []string
var PortErr string
var ProcessesList []string //Kubernetes clusters processes list
var SwapOffDisabled bool   //If this is true the swapOff functionality will be disabled.

const (

	// number of CPUs
	MinCPUs = 2
	// RAM in GiBs
	MinMem = 12
	// Measure of a GiB in terms of bytes
	GB = 1024 * 1024
	// Disk size in GiBs
	MinDisk = 30
	// Disk size in GiBs
	MinAvailDisk = 15

	CheckPass       = "PASS"
	CheckFail       = "FAIL"
	Invalid         = "Invalid"
	Valid           = "Valid"
	InvalidPassword = "Sorry, try again."
)

var (
	// Constants for check failure messages
	PyCliErr                = "Earlier version of pf9ctl already exists. This must be uninstalled."
	ExisitngInstallationErr = "Platform9 packages already exist. These must be uninstalled."
	SudoErr                 = "User running pf9ctl must have privilege (sudo) mode enabled."
	OSPackagesErr           = "Some OS packages needed for the CLI not found"
	CPUErr                  = "At least 2 CPUs are needed on host."
	DiskErr                 = "At least 30 GB of total disk space and 15 GB of free space is needed on host."
	MemErr                  = "At least 12 GB of memory is needed on host."
)

var (
	homeDir, _ = os.UserHomeDir()
	// PyCliPath is the path of virtual env directory of the Python CLI
	PyCliPath = filepath.Join(homeDir, "pf9/pf9-venv")
	// PyCliLink is the Symlink of the Python CLI
	PyCliLink      = "/usr/bin/pf9ctl"
	Centos         = "centos"
	Redhat         = "red hat"
	Ubuntu         = "ubuntu"
	CertsExpireErr = "certificate has expired or is not yet valid"

	//Pf9Dir is the base pf9dir
	Pf9Dir = filepath.Join(homeDir, "pf9")
	//Pf9LogDir is the base path for creating log dir
	Pf9LogDir = filepath.Join(Pf9Dir, "log")
	// Pf9DBDir is the base dir for storing pf9 db config
	Pf9DBDir = filepath.Join(Pf9Dir, "db")
	// Pf9DBLoc represents location of the config file.
	Pf9DBLoc = filepath.Join(Pf9DBDir, "config.json")
	// Pf9Log represents location of the log.
	Pf9Log = filepath.Join(Pf9LogDir, "pf9ctl.log")
	// WaitPeriod is the sleep period for the cli
	// before it starts with the operations.
	WaitPeriod = time.Duration(60)

	VarDir    = "/var/log/pf9"
	EtcDir    = "/etc/pf9"
	Pf9LogLoc = "pf9/log"
	Pf9DirLoc = filepath.Join(homeDir, "/")

	Confidential = []string{"--password", "--user-token"}
)

func init() {
	RequiredPorts = []string{"443", "2379", "2380", "8285", "10250", "10255", "4194", "8285", "3306"}
	ProcessesList = []string{"kubelet", "kube-proxy", "kube-apiserver", "kube-scheduler", "kube-controller"}
	Pf9Packages = []string{"pf9-hostagent", "pf9-comms", "pf9-kube", "pf9-muster"}
}

//These are the constants needed for everything version related
const (
	Version         string = "pf9ctl version: v1.8"
	AWSBucketName   string = "pmkft-assets"
	AWSBucketKey    string = "pf9ctl"
	AWSBucketRegion string = "us-west-1"
	BucketPath      string = "https://" + AWSBucketName + ".s3." + AWSBucketRegion + ".amazonaws.com/" + AWSBucketKey + "_setup"
)
