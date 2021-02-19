package debian

import (
	"fmt"
	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/platform"
	"github.com/platform9/pf9ctl/pkg/util"
	"go.uber.org/zap"
	"math"
	"strconv"
	"strings"
)

// Debian represents debian based host machine
type Debian struct {
	exec cmdexec.Executor
}

// NewDebian creates and returns a new instance of Debian
func NewDebian(exec cmdexec.Executor) *Debian {
	return &Debian{exec}
}

// Check inspects if a host machine meets all the requirements to be a cluster node
func (d *Debian) Check() []platform.Check {
	var checks []platform.Check

	result, err := d.removePyCli()
	checks = append(checks, platform.Check{"Removal of existing CLI", result, err})

	result, err = d.checkExistingInstallation()
	checks = append(checks, platform.Check{"Existing Installation Check", result, err})

	result, err = d.checkOSPackages()
	checks = append(checks, platform.Check{"OS Packages Check", result, err})

	result, err = d.checkSudo()
	checks = append(checks, platform.Check{"SudoCheck", result, err})

	result, err = d.checkCPU()
	checks = append(checks, platform.Check{"CPUCheck", result, err})

	result, err = d.checkDisk()
	checks = append(checks, platform.Check{"DiskCheck", result, err})

	result, err = d.checkMem()
	checks = append(checks, platform.Check{"MemoryCheck", result, err})

	result, err = d.checkPort()
	checks = append(checks, platform.Check{"PortCheck", result, err})

	return checks
}

func (d *Debian) checkExistingInstallation() (bool, error) {

	out, err := d.exec.RunWithStdout("bash", "-c", "dpkg -l | { grep -i 'pf9-' || true; }")
	if err != nil {
		return false, err
	}

	return out == "", nil
}

func (d *Debian) checkOSPackages() (bool, error) {

	packages := []string{"ntp", "curl"}
	errLines := []string{"Packages not found: "}

	for _, p := range packages {
		err := d.exec.Run("bash", "-c", fmt.Sprintf("dpkg -l %s", p))
		if err != nil {
			errLines = append(errLines, p)
		}
	}

	if len(errLines) > 1 {
		return false, fmt.Errorf(strings.Join(errLines, " "))
	}
	return true, nil
}

func (d *Debian) checkSudo() (bool, error) {
	idS, err := d.exec.RunWithStdout("bash", "-c", "id -u | tr -d '\\n'")
	if err != nil {
		return false, err
	}

	id, err := strconv.Atoi(idS)
	if err != nil {
		return false, err
	}

	return id == 0, nil
}

func (d *Debian) checkCPU() (bool, error) {
	cpuS, err := d.exec.RunWithStdout("bash", "-c", "grep -c ^processor /proc/cpuinfo | tr -d '\\n'")
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

func (d *Debian) checkMem() (bool, error) {
	memS, err := d.exec.RunWithStdout("bash", "-c", "echo $(($(getconf _PHYS_PAGES) * $(getconf PAGE_SIZE) / (1024 * 1024))) | tr -d '\\n'")
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

func (d *Debian) checkDisk() (bool, error) {
	diskS, err := d.exec.RunWithStdout("bash", "-c", "df -k . --output=size | sed 1d | xargs | tr -d '\\n'")
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

	availS, err := d.exec.RunWithStdout("bash", "-c", "df -k . --output=avail | sed 1d | xargs | tr -d '\\n'")
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

func (d *Debian) checkPort() (bool, error) {
	var arg string

	// For remote execution the command is wrapped under quotes ("") which creates
	// problems for the awk command. To resolve this, $4 is escaped.
	// Tweaks like this can be prevented by modifying the remote executor.
	switch d.exec.(type) {
	case cmdexec.LocalExecutor:
		arg = "netstat -tupna | awk '{print $4}' | sed -e 's/.*://' | sort | uniq"
	case *cmdexec.RemoteExecutor:
		arg = "netstat -tupna | awk '{print \\$4}' | sed -e 's/.*://' | sort | uniq"
	}

	openPorts, err := d.exec.RunWithStdout("bash", "-c", arg)
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

func (d *Debian) removePyCli() (bool, error) {

	if _, err := d.exec.RunWithStdout("rm", "-rf", util.PyCliPath); err != nil {
		return false, err
	}
	zap.S().Debug("Removed Python CLI directory")

	return true, nil
}

func (d *Debian) Version() (string, error) {
	//using cat command content of os-release file is printed on terminal
	//using grep command os name and version are searched (pretty_name)
	//using cut command required field is selected
	//in this case (PRETTY_NAME="Ubuntu 18.04.2 LTS") second field(18.04.2) is selected using (cut -d ' ' -f 2) command
	out, err := d.exec.RunWithStdout(
		"bash",
		"-c",
		"cat /etc/*os-release | grep -i pretty_name | cut -d ' ' -f 2")
	if err != nil {
		return "", fmt.Errorf("Couldn't read the OS configuration file os-release: %s", err.Error())
	}
	if strings.Contains(string(out), "16") || strings.Contains(string(out), "18") {
		return "debian", nil
	}
	return "", fmt.Errorf("Unable to determine OS type: %s", string(out))
}
