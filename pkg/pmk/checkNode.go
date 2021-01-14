// Copyright © 2020 The Platform9 Systems Inc.

package pmk

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/util"
	"go.uber.org/zap"
)

const (
	// number of CPUs
	minCPUs = 2
	// RAM in GiBs
	minMem = 12
	// Measure of a GiB in terms of bytes
	gB = 1024 * 1024
	// Disk size in GiBs
	minDisk = 30
	// Disk size in GiBs
	minAvailDisk = 15

	checkPass = "PASS"
	checkFail = "FAIL"
)

var requiredPorts = []string{"443", "2379", "2380", "8285", "10250", "10255", "4194", "8285"}

// CheckNode checks the prerequisites for k8s stack
func CheckNode(allClients Client) (bool, error) {

	zap.S().Debug("Received a call to check node.")

	var checks []NodeCheck
	checks = append(checks, osCheck{}, packagesCheck{}, sudoCheck{}, cpuCheck{}, memoryCheck{}, diskCheck{}, portCheck{})

	for _, check := range checks {
		result, err := check.check(allClients.Executor)
		if err != nil {
			zap.S().Errorf("Unable to complete %s: %s ", check.name(), err)
			return false, err
		}

		if result {
			zap.S().Infof("%s : %s", check.name(), checkPass)
		} else {
			zap.S().Infof("%s : %s", check.name(), checkFail)
			zap.S().Debug(check.errMessage())
			return false, nil
		}
	}

	return true, nil
}

// NodeCheck declares contract for all checks
type NodeCheck interface {
	name() string
	check(exec cmdexec.Executor) (bool, error)
	errMessage() string
}

type osCheck struct{}

func (c osCheck) name() string {
	return "OSCheck"
}

func (c osCheck) check(exec cmdexec.Executor) (bool, error) {
	os, err := validatePlatform(exec)
	if err != nil {
		return false, err
	}

	zap.S().Debug("OS running: ", os)

	if os != "redhat" && os != "debian" {
		return false, nil
	}

	return true, nil
}

func (c osCheck) errMessage() string {
	return fmt.Sprint("Only Ubuntu and CentOS VMs are supported")
}

type packagesCheck struct{}

func (c packagesCheck) name() string {
	return "PackagesCheck"
}

func (c packagesCheck) check(exec cmdexec.Executor) (bool, error) {
	os, err := validatePlatform(exec)
	if err != nil {
		return false, err
	}

	present := pf9PackagesPresent(os, exec)

	return !present, nil
}

func (c packagesCheck) errMessage() string {
	return fmt.Sprint("PF9 package are already present on this machine")
}

type sudoCheck struct{}

func (c sudoCheck) name() string {
	return "SudoCheck"
}

func (c sudoCheck) check(exec cmdexec.Executor) (bool, error) {
	idS, err := exec.RunWithStdout("bash", "-c", "id -u | tr -d '\\n'")
	if err != nil {
		return false, err
	}

	id, err := strconv.Atoi(idS)
	if err != nil {
		zap.S().Info(">", err)
		return false, err
	}

	return id == 0, nil
}

func (c sudoCheck) errMessage() string {
	return fmt.Sprint("You need to run this command with sudo")
}

type cpuCheck struct{}

func (c cpuCheck) name() string {
	return "CPUCheck"
}

func (c cpuCheck) check(exec cmdexec.Executor) (bool, error) {
	cpuS, err := exec.RunWithStdout("bash", "-c", "grep -c ^processor /proc/cpuinfo | tr -d '\\n'")
	if err != nil {
		return false, err
	}

	cpu, err := strconv.Atoi(cpuS)
	if err != nil {
		return false, err
	}

	zap.S().Debug("Number of CPUs found: ", cpu)

	return cpu >= minCPUs, nil
}

func (c cpuCheck) errMessage() string {
	return fmt.Sprint("Required minimum %d number of CPUs", minCPUs)
}

type memoryCheck struct{}

func (c memoryCheck) name() string {
	return "MemoryCheck"
}

func (c memoryCheck) check(exec cmdexec.Executor) (bool, error) {
	memS, err := exec.RunWithStdout("bash", "-c", "echo $(($(getconf _PHYS_PAGES) * $(getconf PAGE_SIZE) / (1024 * 1024))) | tr -d '\\n'")
	if err != nil {
		return false, err
	}

	mem, err := strconv.Atoi(memS)
	if err != nil {
		return false, err
	}

	zap.S().Debug("Total memory allocated in GiBs", mem)

	return mem/1024 >= minMem, nil
}

func (c memoryCheck) errMessage() string {
	return fmt.Sprint("Required minimum %d GiB of total memory space", minMem)
}

type diskCheck struct{}

func (c diskCheck) name() string {
	return "DiskCheck"
}

func (c diskCheck) check(exec cmdexec.Executor) (bool, error) {
	diskS, err := exec.RunWithStdout("bash", "-c", "df -k . --output=size | sed 1d | xargs | tr -d '\\n'")
	if err != nil {
		return false, err
	}

	disk, err := strconv.ParseFloat(diskS, 32)
	if err != nil {
		return false, err
	}

	if math.Ceil(disk/gB) < minDisk {
		return false, nil
	}

	zap.S().Debug("Total disk space: ", disk)

	availS, err := exec.RunWithStdout("bash", "-c", "df -k . --output=avail | sed 1d | xargs | tr -d '\\n'")
	if err != nil {
		return false, err
	}

	avail, err := strconv.ParseFloat(availS, 32)
	if err != nil {
		return false, err
	}

	zap.S().Debug("Available disk space: ", avail)

	return math.Ceil(avail/gB) >= minAvailDisk, nil
}

func (c diskCheck) errMessage() string {
	return fmt.Sprintf("Required minimum %d GiB of total disk space and %d GiB of free disk space", minDisk, minAvailDisk)
}

type portCheck struct{}

func (c portCheck) name() string {
	return "PortCheck"
}

func (c portCheck) check(exec cmdexec.Executor) (bool, error) {
	openPorts, err := exec.RunWithStdout("bash", "-c", "netstat -tupna | awk '{print $4}' | sed -e 's/.*://' | sort | uniq")
	if err != nil {
		return false, err
	}

	openPortsArray := strings.Split(string(openPorts), "\n")

	intersection := util.Intersect(requiredPorts, openPortsArray)

	if len(intersection) != 0 {
		zap.S().Debug("Ports required but not available: ", intersection)
		return false, nil
	}

	return true, nil
}

func (c portCheck) errMessage() string {
	return fmt.Sprintf("Required minimum %d GiB of total disk space and %d GiB of free disk space", minDisk, minAvailDisk)
}
