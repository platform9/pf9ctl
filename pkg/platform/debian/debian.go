package debian

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/platform"
	"github.com/platform9/pf9ctl/pkg/swapoff"
	"github.com/platform9/pf9ctl/pkg/util"
	"go.uber.org/zap"
)

var (
	packages                   = []string{"curl", "uuid-runtime", "net-tools", "jq"}
	packageInstallError        = "Packages not found and could not be installed"
	MissingPkgsInstalledDebian bool
	k8sPresentError            = errors.New("A Kubernetes cluster is already running on node")
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
	checks = append(checks, platform.Check{"Removal of existing CLI", false, result, err, util.PyCliErr})

	result, err = d.checkExistingInstallation()
	checks = append(checks, platform.Check{"Existing Platform9 Packages Check", true, result, err, util.ExisitngInstallationErr})

	result, err = d.checkOSPackages()
	checks = append(checks, platform.Check{"Required OS Packages Check", true, result, err, fmt.Sprintf("%s. %s", util.OSPackagesErr, err)})

	result, err = d.checkSudo()
	checks = append(checks, platform.Check{"SudoCheck", true, result, err, util.SudoErr})

	result, err = d.checkCPU()
	checks = append(checks, platform.Check{"CPUCheck", false, result, err, fmt.Sprintf("%s %s", util.CPUErr, err)})

	result, err = d.checkDisk()
	checks = append(checks, platform.Check{"DiskCheck", false, result, err, fmt.Sprintf("%s %s", util.DiskErr, err)})

	result, err = d.checkMem()
	checks = append(checks, platform.Check{"MemoryCheck", false, result, err, fmt.Sprintf("%s %s", util.MemErr, err)})

	result, err = d.checkPort()
	checks = append(checks, platform.Check{"PortCheck", true, result, err, fmt.Sprintf("%s", err)})

	result, err = d.checkKubernetesCluster()
	checks = append(checks, platform.Check{"Existing Kubernetes Cluster Check", true, result, err, fmt.Sprintf("%s", err)})

	result, err = d.checkNoexecPermission()
	checks = append(checks, platform.Check{"Check exec permission on /tmp", true, result, err, fmt.Sprintf("%s", err)})

	result, err = d.checkIfdpkgISLock()
	checks = append(checks, platform.Check{"Check lock on dpkg", true, result, err, fmt.Sprintf("%s", err)})

	result, err = d.checkIfaptISLock()
	checks = append(checks, platform.Check{"Check lock on apt", true, result, err, fmt.Sprintf("%s", err)})

	result, err = d.checkPIDofSystemd()
	checks = append(checks, platform.Check{"Check if system is booted with systemd", true, result, err, fmt.Sprintf("%s", err)})

	result, err = d.checkIfTimesyncServiceRunning()
	checks = append(checks, platform.Check{"Check time synchronization", true, result, err, fmt.Sprintf("%s", err)})

	if !util.SwapOffDisabled {
		result, err = d.disableSwap()
		checks = append(checks, platform.Check{"Disabling swap and removing swap in fstab", true, result, err, fmt.Sprintf("%s", err)})
	}
	return checks
}

func (d *Debian) checkKubernetesCluster() (bool, error) {
	for _, proc := range util.ProcessesList {
		//Checking if kubernetes process is running on the host or not
		_, err := d.exec.RunWithStdout("bash", "-c", fmt.Sprintf("ps -A | grep -i %s", proc))

		if err != nil {
			return true, nil
		} else if d.checkDocker(); err != nil {
			return true, nil
		} else {
			return false, k8sPresentError
		}
	}
	return true, nil
}

func (d *Debian) checkDocker() error {
	//Checking kube-proxy. Every node in kubernetes cluster runs kube-proxy.
	var err error
	for _, proc := range util.ProcessesList {
		_, err = d.exec.RunWithStdout("bash", "-c", fmt.Sprintf("docker ps | grep -i %s", proc))
		if err == nil {
			return k8sPresentError
		}
	}
	return err
}

func (d *Debian) checkExistingInstallation() (bool, error) {

	var (
		out string
		err error
	)
	for _, p := range util.Pf9Packages {
		cmd := fmt.Sprintf("dpkg -l | { grep -i '%s' || true; }", p)
		out, err = d.exec.RunWithStdout("bash", "-c", cmd)
		if err != nil {
			return false, err
		}
		if out != "" {
			return false, err
		}
	}
	return true, nil
}

func (d *Debian) checkOSPackages() (bool, error) {
	// This Flag will be set if we install missing packages

	errLines := []string{packageInstallError}

	zap.S().Debug("Checking OS Packages")
	for _, p := range packages {
		err := d.exec.Run("bash", "-c", fmt.Sprintf("dpkg-query -s %s", p))
		if err != nil {
			zap.S().Debugf("Package %s not found, trying to install", p)
			zap.S().Debug("Installing missing packages, this may take a few minutes")
			if err = d.installOSPackages(p); err != nil {
				zap.S().Debugf("Error installing package %s: %s", p, err)
				errLines = append(errLines, p)
			} else {
				MissingPkgsInstalledDebian = true
				zap.S().Debugf("Missing package %s installed", p)
			}
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

	if cpu >= util.MinCPUs {
		return true, nil
	}
	return false, fmt.Errorf("Number of CPUs found: %d", cpu)
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

	if math.Ceil(mem/1024) >= util.MinMem {
		return true, nil
	}
	return false, fmt.Errorf("Total memory found: %.0f GB", math.Ceil(mem/1024))
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
		return false, fmt.Errorf("Disk Space found: %.0f GB", math.Ceil(disk/util.GB))
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

	if math.Ceil(avail/util.GB) >= util.MinAvailDisk {
		return true, nil
	}
	return false, fmt.Errorf("Available disk space: %.0f GB", math.Trunc(avail/util.GB))
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
		ports_list := strings.Join(intersection[:], ", ")
		return false, fmt.Errorf("Following port(s) should not be in use: %s", ports_list)
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
	if strings.Contains(string(out), "16") || strings.Contains(string(out), "18") || strings.Contains(string(out), "20") {
		return "debian", nil
	}
	return "", fmt.Errorf("Unable to determine OS type: %s", string(out))
}

func (d *Debian) installOSPackages(p string) error {
	zap.S().Debug("Trying apt update...")
	_, err := d.exec.RunWithStdout("bash", "-c", "apt update -qq")
	if err != nil {
		return err
	}

	zap.S().Debugf("Trying to install package %s", p)
	_, err = d.exec.RunWithStdout("bash", "-c", fmt.Sprintf("apt install -qq -y %s", p))
	return nil
}

func (d *Debian) disableSwap() (bool, error) {
	err := swapoff.SetupNode(d.exec)
	if err != nil {
		return false, errors.New("error occured while disabling swap")
	} else {
		return true, nil
	}
}

func (d *Debian) checkNoexecPermission() (bool, error) {
	_, err := d.exec.RunWithStdout("bash", "-c", `mount | grep ' /tmp ' | grep 'noexec'`)
	if err != nil {
		return true, nil
	} else {
		return false, errors.New("/tmp is not having exec permission")
	}
}

func (d *Debian) checkIfdpkgISLock() (bool, error) {

	var f = []string{"lock", "lock-frontend"}
	for _, file := range f {
		_, err := d.exec.RunWithStdout("bash", "-c", fmt.Sprintf("lsof /var/lib/dpkg/%s", file))
		if err != nil {
			return true, nil
		} else {
			zap.S().Infof("Unable to acquire the dpkg lock on %s", file)
			return false, fmt.Errorf("Unable to acquire the dpkg")
		}
	}
	return true, fmt.Errorf("Unable to check dpkg lock")
}

func (d *Debian) checkIfaptISLock() (bool, error) {
	_, err := d.exec.RunWithStdout("bash", "-c", "lsof /var/lib/apt/lists/lock")
	if err != nil {
		return true, nil
	} else {
		return false, errors.New("apt is locked")
	}
}

func (d *Debian) checkPIDofSystemd() (bool, error) {
	_, err := d.exec.RunWithStdout("bash", "-c", "ps -p 1 -o comm= | grep systemd")
	if err != nil {
		return false, errors.New("System is not booted with systemd")
	} else {
		return true, nil
	}
}

func (d *Debian) checkIfTimesyncServiceRunning() (bool, error) {
	zap.S().Debug("Checking if systemd-timesyncd service is running")
	err := d.IsSystemdTimesyncdIsRunning()
	if err != nil {
		if _, err := d.exec.RunWithStdout("bash", "-c", "systemctl start systemd-timesyncd"); err != nil {
			zap.S().Debug("failed to start systemd-timesyncd service")
			zap.S().Debug("Checking if ntp service is running")
			err := d.IsNtpRunning()
			if err != nil {
				if _, err := d.exec.RunWithStdout("bash", "-c", "systemctl start ntp"); err != nil {
					zap.S().Debug("failed to start ntp service")
					zap.S().Debug("Checking if chronyd service is running")
					err := d.IsChronydRunning()
					if err != nil {
						err := d.DownloadAndInstallSystemdTimesyncd()
						if err != nil {
							return false, err
						} else {
							return true, nil
						}
					} else {
						zap.S().Debug("chronyd service is running")
						return true, nil
					}
				} else {
					zap.S().Debug("started ntp service")
					return true, nil
				}
			} else {
				zap.S().Debug("ntp service is running")
				return true, nil
			}
		} else {
			zap.S().Debug("started systemd-timesyncd service")
			return true, nil
		}
	} else {
		zap.S().Debug("systemd-timesyncd service is running")
		return true, nil
	}
}

func (d *Debian) IsSystemdTimesyncdIsRunning() error {
	_, err := d.exec.RunWithStdout("bash", "-c", "systemctl status systemd-timesyncd | grep 'active (running)'")
	if err != nil {
		zap.S().Debug("systemd-timesyncd service is not running")
		return errors.New("systemd-timesyncd service is not running")
	} else {
		zap.S().Debug("systemd-timesyncd service is running")
		return nil
	}
}

func (d *Debian) IsNtpRunning() error {
	_, err := d.exec.RunWithStdout("bash", "-c", "systemctl status ntp | grep 'active (running)'")
	if err != nil {
		zap.S().Debug("ntp service is not running")
		return errors.New("ntp service is not running")
	} else {
		zap.S().Debug("ntp service is running")
		return nil
	}
}

func (d *Debian) IsChronydRunning() error {
	_, err := d.exec.RunWithStdout("bash", "-c", "systemctl status chronyd | grep 'active (running)'")
	if err != nil {
		zap.S().Debug("chronyd service is not running")
		return errors.New("chronyd service is not running")
	} else {
		zap.S().Debug("chronyd service is running")
		return nil
	}
}

func (d *Debian) DownloadAndInstallSystemdTimesyncd() error {
	err := d.installOSPackages("systemd-timesyncd")
	if err != nil {
		return errors.New("could not install systemd-timesyncd package")
	} else {
		return nil
	}
}
