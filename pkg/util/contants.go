package util

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

func init() {
	RequiredPorts = []string{"443", "2379", "2380", "8285", "10250", "10255", "4194", "8285"}
}
