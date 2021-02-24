package util

import (
	"fmt"
	"os"
	"path/filepath"
)

var RequiredPorts []string
var PortErr string

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

	CheckPass = "PASS"
	CheckFail = "FAIL"
)

var (
	// Constants for check failure messages
	PyCliErr                = "Earlier version of pf9ctl already exists. This must be uninstalled."
	ExisitngInstallationErr = "Platform9 packages already exist. These must be uninstalled."
	SudoErr                 = "User running pf9ctl must have privilege (sudo) mode enabled."
	OSPackagesErr           = "Some OS packages needed for the CLI not found"
	CPUErr                  = "At least 2 CPUs are needed on host."
	DiskErr                 = "At least 30 GB of disk space is needed on host."
	MemErr                  = "At least 12 GB of memory is needed on host."
)

var (
	homeDir, _ = os.UserHomeDir()
	// PyCliPath is the path of virtual env directory of the Python CLI
	PyCliPath = filepath.Join(homeDir, "pf9/pf9-venv")
	// PyCliLink is the Symlink of the Python CLI
	PyCliLink      = "/usr/bin/pf9ctl"
	Centos         = "centos"
	Redhat         = "redhat"
	Ubuntu         = "ubuntu"
	CertsExpireErr = "certificate has expired or is not yet valid"
)

func init() {
	RequiredPorts = []string{"443", "2379", "2380", "8285", "10250", "10255", "4194", "8285"}
	PortErr = fmt.Sprintf("Ports required to be available %s", RequiredPorts)
}
