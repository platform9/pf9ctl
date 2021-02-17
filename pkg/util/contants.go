package util

import (
	"os"
	"path/filepath"
)

var RequiredPorts []string

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
	homeDir, _ = os.UserHomeDir()
	// PyCliPath is the path of virtual env directory of the Python CLI
	PyCliPath = filepath.Join(homeDir, "pf9/pf9-venv")
	// PyCliLink is the Symlink of the Python CLI
	PyCliLink = "/usr/bin/pf9ctl"
	Centos = "centos"
	Redhat = "redhat"
	Ubuntu = "ubuntu"
)

func init() {
	RequiredPorts = []string{"443", "2379", "2380", "8285", "10250", "10255", "4194", "8285"}
}
