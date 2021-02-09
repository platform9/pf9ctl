package centos

import (
	"math"
	"strconv"
	"strings"

	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/platform"
	"github.com/platform9/pf9ctl/pkg/util"
	"go.uber.org/zap"
)

// CentOS reprents centos based host machine
type CentOS struct {
	exec cmdexec.Executor
}

// NewCentOS creates and returns a new instance of CentOS
func NewCentOS(exec cmdexec.Executor) *CentOS {
	return &CentOS{exec}
}

// Check inspects if a host machine meets all the requirements to be a cluster node
func (c *CentOS) Check() []platform.Check {
	var checks []platform.Check

	result, err := c.removePyCli()
	checks = append(checks, platform.Check{"Python CLI Removal", result, err})

	result, err = c.checkPackages()
	checks = append(checks, platform.Check{"Existing Installation Check", result, err})

	result, err = c.checkSudo()
	checks = append(checks, platform.Check{"SudoCheck", result, err})

	result, err = c.checkCPU()
	checks = append(checks, platform.Check{"CPUCheck", result, err})

	result, err = c.checkDisk()
	checks = append(checks, platform.Check{"DiskCheck", result, err})

	result, err = c.checkMem()
	checks = append(checks, platform.Check{"MemoryCheck", result, err})

	result, err = c.checkPort()
	checks = append(checks, platform.Check{"PortCheck", result, err})

	return checks
}

func (c *CentOS) checkPackages() (bool, error) {
	var err error
	err = c.exec.Run("bash", "-c", "yum list installed | grep -i 'pf9-'")

	return !(err == nil), nil
}

func (c *CentOS) checkSudo() (bool, error) {
	idS, err := c.exec.RunWithStdout("bash", "-c", "id -u | tr -d '\\n'")
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

func (c *CentOS) checkCPU() (bool, error) {
	cpuS, err := c.exec.RunWithStdout("bash", "-c", "grep -c ^processor /proc/cpuinfo | tr -d '\\n'")
	if err != nil {
		return false, err
	}

	cpu, err := strconv.Atoi(cpuS)
	if err != nil {
		return false, err
	}

	zap.S().Debug("Number of CPUs found: ", cpu)

	return cpu >= util.MinCPUs, nil
}

func (c *CentOS) checkMem() (bool, error) {
	memS, err := c.exec.RunWithStdout("bash", "-c", "echo $(($(getconf _PHYS_PAGES) * $(getconf PAGE_SIZE) / (1024 * 1024))) | tr -d '\\n'")
	if err != nil {
		return false, err
	}

	mem, err := strconv.ParseFloat(memS, 32)
	if err != nil {
		return false, err
	}

	zap.S().Debug("Total memory allocated in GiBs", mem)

	return math.Ceil(mem/1024) >= util.MinMem, nil
}

func (c *CentOS) checkDisk() (bool, error) {
	diskS, err := c.exec.RunWithStdout("bash", "-c", "df -k . --output=size | sed 1d | xargs | tr -d '\\n'")
	if err != nil {
		return false, err
	}

	disk, err := strconv.ParseFloat(diskS, 32)
	if err != nil {
		return false, err
	}

	if math.Ceil(disk/util.GB) < util.MinDisk {
		return false, nil
	}

	zap.S().Debug("Total disk space: ", disk)

	availS, err := c.exec.RunWithStdout("bash", "-c", "df -k . --output=avail | sed 1d | xargs | tr -d '\\n'")
	if err != nil {
		return false, err
	}

	avail, err := strconv.ParseFloat(availS, 32)
	if err != nil {
		return false, err
	}

	zap.S().Debug("Available disk space: ", avail)

	return math.Ceil(avail/util.GB) >= util.MinAvailDisk, nil
}

func (c *CentOS) checkPort() (bool, error) {
	var arg string

	// For remote execution the command is wrapped under quotes ("") which creates
	// problems for the awk command. To resolve this, $4 is escaped.
	// Tweaks like this can be prevented by modifying the remote executor.
	switch c.exec.(type) {
	case cmdexec.LocalExecutor:
		arg = "netstat -tupna | awk '{print $4}' | sed -e 's/.*://' | sort | uniq"
	case *cmdexec.RemoteExecutor:
		arg = "netstat -tupna | awk '{print \\$4}' | sed -e 's/.*://' | sort | uniq"
	}

	openPorts, err := c.exec.RunWithStdout("bash", "-c", arg)
	if err != nil {
		return false, err
	}

	openPortsArray := strings.Split(string(openPorts), "\n")

	intersection := util.Intersect(util.RequiredPorts, openPortsArray)

	if len(intersection) != 0 {
		zap.S().Debug("Ports required but not available: ", intersection)
		return false, nil
	}

	return true, nil
}

func (c *CentOS) removePyCli() (bool, error) {

	_, err := c.exec.RunWithStdout("ls", util.PyCliLink)
	if err == nil {
		if _, err = c.exec.RunWithStdout("rm", "-rf", util.PyCliLink); err != nil {
			return false, err
		}
		zap.S().Debug("Removed Python CLI symlink")
	}

	_, err = c.exec.RunWithStdout("ls", util.PyCliPath)
	if err == nil {
		if _, err = c.exec.RunWithStdout("rm", "-rf", util.PyCliPath); err != nil {
			return false, err
		}
		zap.S().Debug("Removed Python CLI directory")
	}

	return true, nil
}
