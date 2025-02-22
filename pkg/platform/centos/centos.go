package centos

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/platform"
	"github.com/platform9/pf9ctl/pkg/swapoff"
	"github.com/platform9/pf9ctl/pkg/util"
	"go.uber.org/zap"
)

var (
	packages                   = []string{"ntp", "curl", "policycoreutils", "policycoreutils-python", "selinux-policy", "selinux-policy-targeted", "libselinux-utils", "net-tools"}
	packageInstallError        = "Packages not found and could not be installed"
	MissingPkgsInstalledCentos bool
	centos                     bool
	version                    string
	k8sPresentError            = errors.New("A Kubernetes cluster is already running on node")
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
	checks = append(checks, platform.Check{"Removal of existing CLI", false, result, err, util.PyCliErr})

	result, err = c.CheckExistingInstallation()
	checks = append(checks, platform.Check{"Existing Platform9 Packages Check", false, result, err, util.ExisitngInstallationErr})

	result, err = c.checkOSPackages()
	checks = append(checks, platform.Check{"Required OS Packages Check", true, result, err, fmt.Sprintf("%s. %s", util.OSPackagesErr, err)})

	result, err = c.checkSudo()
	checks = append(checks, platform.Check{"SudoCheck", true, result, err, util.SudoErr})

	result, err = c.checkEnabledRepos()
	checks = append(checks, platform.Check{"Required Enabled Repositories Check", true, result, err, fmt.Sprintf("%s", err)})

	result, err = c.checkCPU()
	checks = append(checks, platform.Check{"CPUCheck", false, result, err, fmt.Sprintf("%s %s", util.CPUErr, err)})

	result, err = c.checkDisk()
	checks = append(checks, platform.Check{"DiskCheck", false, result, err, fmt.Sprintf("%s %s", util.DiskErr, err)})

	result, err = c.checkMem()
	checks = append(checks, platform.Check{"MemoryCheck", false, result, err, fmt.Sprintf("%s %s", util.MemErr, err)})

	result, err = c.checkPort()
	checks = append(checks, platform.Check{"PortCheck", false, result, err, fmt.Sprintf("%s", err)})

	result, err = c.CheckKubernetesCluster()
	checks = append(checks, platform.Check{"Existing Kubernetes Cluster Check", false, result, err, fmt.Sprintf("%s", err)})

	result, err = c.checkPIDofSystemd()
	checks = append(checks, platform.Check{"Check if system is booted with systemd", true, result, err, fmt.Sprintf("%s", err)})

	result, err = c.checkFirewalldIsRunning()
	checks = append(checks, platform.Check{"Check if firewalld service is not running", false, result, err, fmt.Sprintf("%s", err)})

	if !util.SwapOffDisabled {
		result, err = c.disableSwap()
		checks = append(checks, platform.Check{"Disabling swap and removing swap in fstab", true, result, err, fmt.Sprintf("%s", err)})
	}

	return checks
}

func (c *CentOS) CheckKubernetesCluster() (bool, error) {
	for _, proc := range util.ProcessesList {
		//Checking if kubernetes process is running on the host or not
		_, err := c.exec.RunWithStdout("bash", "-c", fmt.Sprintf("ps -A | grep -i %s", proc))

		if err != nil {
			return true, nil
		} else if c.checkDocker(); err != nil {
			return true, nil
		} else {
			return false, k8sPresentError
		}
	}
	return true, nil
}

func (c *CentOS) checkDocker() error {
	//Checking kube-proxy. Every node in kubernetes cluster runs kube-proxy.
	var err error
	for _, proc := range util.ProcessesList {
		_, err = c.exec.RunWithStdout("bash", "-c", fmt.Sprintf("docker ps | grep -i %s", proc))
		if err == nil {
			return k8sPresentError
		}
	}
	return err
}

func (c *CentOS) CheckExistingInstallation() (bool, error) {

	var (
		out string
		err error
	)
	for _, p := range util.Pf9Packages {
		cmd := fmt.Sprintf("yum list installed | { grep -i '%s' || true; }", p)
		out, err = c.exec.RunWithStdout("bash", "-c", cmd)
		if err != nil {
			return false, err
		}
		if out != "" {
			return false, err
		}
	}
	return true, nil
}

func (c *CentOS) checkOSPackages() (bool, error) {

	var rhel8, rocky9 bool
	errLines := []string{packageInstallError}
	zap.S().Debug("Checking OS Packages")

	rhel8, _ = regexp.MatchString(`.*8\.([5-9]|1[0])\.*`, string(version))
	rocky9, _ = regexp.MatchString(`.*9\.[1-5]\.*`, string(version))

	if platform.SkipOSChecks {
		rhel8, _ = regexp.MatchString(`8\.\d{1,2}`, string(version))
		rocky9, _ = regexp.MatchString(`9\.\d{1,2}`, string(version))
	}
	for _, p := range packages {
		if !centos && (rhel8 || rocky9) {
			switch p {
			case "policycoreutils-python":
				if rhel8 {
					p = "python3-policycoreutils"
				} else if rocky9 {
					p = "policycoreutils-python-utils"
				}

			case "ntp":
				p = "chrony"
			}
		}

		err := c.exec.Run("bash", "-c", fmt.Sprintf("yum list installed %s", p))
		if err != nil {
			zap.S().Debug("Installing missing packages, this may take a few minutes")
			zap.S().Debugf("Package %s not found, trying to install", p)
			if err = c.installOSPackages(p); err != nil {
				zap.S().Debugf("Error installing package %s: %s", p, err)
				errLines = append(errLines, p)
			} else {
				MissingPkgsInstalledCentos = true
				zap.S().Debugf("Missing package %s installed", p)
			}
		}
	}

	if len(errLines) > 1 {
		return false, fmt.Errorf(strings.Join(errLines, " "))
	}
	return true, nil
}

func (c *CentOS) checkEnabledRepos() (bool, error) {

	var centos, rhel8 bool
	centos, _ = regexp.MatchString(`.*7\.[3-9]\.*`, string(version))
	rhel8, _ = regexp.MatchString(`.*8\.([5-9]|1[0])\.*`, string(version))

	if platform.SkipOSChecks {
		centos, _ = regexp.MatchString(`7\.\d{1,2}`, string(version))
		rhel8, _ = regexp.MatchString(`8\.\d{1,2}`, string(version))
	}

	output, err := c.exec.RunWithStdout("bash", "-c", "yum repolist")
	if err != nil {
		zap.S().Debug("Error executing 'yum repolist' command:", err)
		return false, err
	}

	var enable_repos []string
	var command string

	if centos {
		command = "yum-config-manager --enable %s"
		if !strings.Contains(string(output), "base/") {
			enable_repos = append(enable_repos, "base")
		}
		if !strings.Contains(string(output), "extras/") {
			enable_repos = append(enable_repos, "extras")
		}
	} else if rhel8 {
		command = "subscription-manager repos --enable %s"
		if !strings.Contains(string(output), "BaseOS") {
			enable_repos = append(enable_repos, "rhel-8-for-x86_64-baseos-rpms")
		}
		if !strings.Contains(string(output), "AppStream") {
			enable_repos = append(enable_repos, "rhel-8-for-x86_64-appstream-rpms")
		}
	}

	for _, r := range enable_repos {
		err := c.exec.Run("bash", "-c", fmt.Sprintf(command, r))
		zap.S().Debug("Ran command sudo ", `"bash" "-c" `, fmt.Sprintf(command, r))
		if err != nil {
			zap.S().Debug("Error enabling repository: ", r)
			return false, err
		}
	}
	zap.S().Debug("Required repositories are enabled")
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
	diskS, err := c.exec.RunWithStdout("bash", "-c", "df -k / --output=size | sed 1d | xargs | tr -d '\\n'")
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

	availS, err := c.exec.RunWithStdout("bash", "-c", "df -k / --output=avail | sed 1d | xargs | tr -d '\\n'")
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
		ports_list := strings.Join(intersection[:], ", ")
		return false, fmt.Errorf("Following port(s) should not be in use: %s", ports_list)
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
	var cmd string

	_, err := c.exec.RunWithStdout("bash", "-c", "grep -i 'CentOS Linux' /etc/*release")
	if err != nil {
		cmd = fmt.Sprintf("grep -oP '(?<=^VERSION_ID=).+' /etc/os-release")
	} else {
		centos = true
		cmd = fmt.Sprintf("cat /etc/*release | grep 'CentOS Linux release' -m 1 | cut -f4 -d ' '")
	}

	version, err = c.exec.RunWithStdout("bash", "-c", cmd)
	if err != nil {
		return "", fmt.Errorf("Couldn't read the OS configuration file os-release: %s", err.Error())
	}
	if centos {
		//comparing because we are not supporting centos 8.x
		if match, _ := regexp.MatchString(`.*7\.[3-9]\.*`, string(version)); match {
			return "redhat", nil
		}
	}
	if match, _ := regexp.MatchString(`.*7\.[3-9]\.*|.*8\.([5-9]|1[0])\.*|.*9\.[1-5]\.*`, string(version)); match {
		return "redhat", nil
	}
	return "", fmt.Errorf("Unable to determine OS type: %s", string(version))

}

func (c *CentOS) installOSPackages(p string) error {
	zap.S().Debug("Trying yum update...")
	_, err := c.exec.RunWithStdout("bash", "-c", "yum clean all -q")
	if err != nil {
		return err
	}

	zap.S().Debugf("Trying to install package %s", p)
	_, err = c.exec.RunWithStdout("bash", "-c", fmt.Sprintf("yum -q -y install %s", p))
	if err != nil {
		return err
	}

	switch p {
	case "chrony":
		{
			_, err = c.exec.RunWithStdout("bash", "-c", "systemctl start chronyd")
			if err != nil {
				zap.S().Debug("Failed to start chronyd time sync service")
			} else {
				zap.S().Debug("chronyd time sync service started")
			}
		}
	case "ntp":
		{
			_, err = c.exec.RunWithStdout("bash", "-c", "systemctl start ntpd")
			if err != nil {
				zap.S().Debug("Failed to start ntpd time sync service")
			} else {
				zap.S().Debug("ntpd time sync service started")
			}
		}
	}
	return nil
}

func (c *CentOS) disableSwap() (bool, error) {
	err := swapoff.SetupNode(c.exec)
	if err != nil {
		return false, errors.New("error occurred while disabling swap")
	} else {
		return true, nil
	}
}

func (c *CentOS) checkPIDofSystemd() (bool, error) {
	_, err := c.exec.RunWithStdout("bash", "-c", "ps -p 1 -o comm= | grep systemd")
	if err != nil {
		return false, errors.New("System is not booted with systemd")
	} else {
		return true, nil
	}
}

func (c *CentOS) checkFirewalldIsRunning() (bool, error) {
	_, err := c.exec.RunWithStdout("bash", "-c", "systemctl is-active firewalld")
	if err != nil {
		return true, nil
	} else {
		return false, errors.New("firewalld service is running")
	}
}
