package centos

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/platform"
	"github.com/platform9/pf9ctl/pkg/util"
	"go.uber.org/zap"
)

var (
	packages            = []string{"ntp", "curl"}
	packageInstallError = "Packages not found and could not be installed"
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
	checks = append(checks, platform.Check{"Removal of existing CLI", result, err, util.PyCliErr})

	result, err = c.checkExistingInstallation()
	checks = append(checks, platform.Check{"Existing Installation Check", result, err, util.ExisitngInstallationErr})

	result, err = c.checkOSPackages()
	checks = append(checks, platform.Check{"OS Packages Check", result, err, fmt.Sprintf("%s. %s", util.OSPackagesErr, err)})

	result, err = c.checkSudo()
	checks = append(checks, platform.Check{"SudoCheck", result, err, util.SudoErr})

	result, err = c.checkCPU()
	checks = append(checks, platform.Check{"CPUCheck", result, err, fmt.Sprintf("%s %s", util.CPUErr, err)})

	result, err = c.checkDisk()
	checks = append(checks, platform.Check{"DiskCheck", result, err, fmt.Sprintf("%s %s", util.DiskErr, err)})

	result, err = c.checkMem()
	checks = append(checks, platform.Check{"MemoryCheck", result, err, fmt.Sprintf("%s %s", util.MemErr, err)})

	result, err = c.checkPort()
	checks = append(checks, platform.Check{"PortCheck", result, err, util.PortErr})

	return checks
}

func (c *CentOS) checkExistingInstallation() (bool, error) {

	out, err := c.exec.RunWithStdout("bash", "-c", "yum list installed | { grep -i 'pf9-' || true; }")
	if err != nil {
		return false, nil
	}

	return out == "", nil
}

func (c *CentOS) checkOSPackages() (bool, error) {

	errLines := []string{packageInstallError}
	zap.S().Info("Checking OS Packages")

	for _, p := range packages {
		err := c.exec.Run("bash", "-c", fmt.Sprintf("yum list installed %s", p))
		if err != nil {
			zap.S().Info("Installing missing packages, this may take a few minutes")
			zap.S().Debugf("Package %s not found, trying to install", p)
			if err = c.installOSPackages(p); err != nil {
				zap.S().Debugf("Error installing package %s: %s", p, err)
				errLines = append(errLines, p)
			} else {
				zap.S().Infof("Missing package %s installed", p)
			}
		}
	}

	if len(errLines) > 1 {
		return false, fmt.Errorf(strings.Join(errLines, " "))
	}
	return true, nil
}

func (c *CentOS) checkSudo() (bool, error) {
	idS, err := c.exec.RunWithStdout("bash", "-c", "id -u | tr -d '\\n'")
	if err != nil {
		return false, err
	}

	id, err := strconv.Atoi(idS)
	if err != nil {
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

	if cpu >= util.MinCPUs {
		return true, nil
	}
	return false, fmt.Errorf("Number of CPUs found: %d", cpu)
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

	if math.Ceil(mem/1024) >= util.MinMem {
		return true, nil
	}
	return false, fmt.Errorf("Total memory found: %.0f GB", math.Ceil(mem/1024))
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
		return false, fmt.Errorf("Disk Space found: %.0f GB", math.Ceil(disk/util.GB))
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

	if math.Ceil(avail/util.GB) >= util.MinAvailDisk {
		return true, nil
	}
	return false, fmt.Errorf("Available disk space: %.0f GB", math.Trunc(avail/util.GB))
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

	if _, err := c.exec.RunWithStdout("rm", "-rf", util.PyCliPath); err != nil {
		return false, err
	}
	zap.S().Debug("Removed Python CLI directory")

	return true, nil
}

func (c *CentOS) Version() (string, error) {
	//using cat command content of os-release file is printed on terminal
	//using grep command os name and version are searched. e.g (CentOS Linux release 7.6.1810 (Core))
	//using cut command required field (7.6.1810) is selected.
	out, err := c.exec.RunWithStdout(
		"bash",
		"-c",
		"cat /etc/*release | grep '(Core)' | grep 'CentOS Linux release' -m 1 | cut -f4 -d ' '")
	if err != nil {
		return "", fmt.Errorf("Couldn't read the OS configuration file os-release: %s", err.Error())
	}
	if match, _ := regexp.MatchString(`.*7\.[3-9]\.*`, string(out)); match {
		return "redhat", nil
	}
	return "", fmt.Errorf("Unable to determine OS type: %s", string(out))

}

func (c *CentOS) installOSPackages(p string) error {
	zap.S().Debug("Trying yum update...")
	_, err := c.exec.RunWithStdout("bash", "-c", "yum update -y -q")
	if err != nil {
		return err
	}

	zap.S().Debugf("Trying to install package %s", p)
	_, err = c.exec.RunWithStdout("bash", "-c", fmt.Sprintf("yum -q -y install %s", p))
	return nil
}
