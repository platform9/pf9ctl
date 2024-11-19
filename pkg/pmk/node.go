// Copyright © 2020 The Platform9 Systems Inc.
package pmk

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/platform9/pf9ctl/pkg/client"
	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/keystone"
	"github.com/platform9/pf9ctl/pkg/objects"
	"github.com/platform9/pf9ctl/pkg/platform"
	"github.com/platform9/pf9ctl/pkg/platform/centos"
	"github.com/platform9/pf9ctl/pkg/platform/debian"
	"github.com/platform9/pf9ctl/pkg/util"
	"go.uber.org/zap"
)

// This variable is assigned with StatusCode during hostagent installation
var HostAgent int
var IsRemoteExecutor bool
var homeDir string
var isCDU bool

const (
	// Response Status Codes
	HostAgentCertless = 200
	HostAgentLegacy   = 404
)

// Sends an event to segment based on the input string and uses auth as keystone UUID property.
func sendSegmentEvent(allClients client.Client, eventStr string, auth keystone.KeystoneAuth, isError bool) {

	var errorStr, suffixStr, status string

	if isError {
		status = "FAIL"
		errorStr = eventStr
		suffixStr = "Prep-node : ERROR"
	} else {
		status = "PASS"
		errorStr = ""
		suffixStr = "Prep-node : " + eventStr
	}

	if err := allClients.Segment.SendEvent(suffixStr, auth, status, errorStr); err != nil {
		zap.S().Debugf("Unable to send Segment event for Node prep. Error: %s", err.Error())
	}

	// Close the segment for error path. Cmd level closure does not work due to FatalF.
	if isError {
		defer allClients.Segment.Close()
	}
}

// PrepNode sets up prerequisites for k8s stack
func PrepNode(ctx objects.Config, allClients client.Client, auth keystone.KeystoneAuth) error {
	// Building our new spinner
	isCDU = false
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Color("red")

	zap.S().Debug("Received a call to start preparing node(s).")
	s.Start() // Start the spinner
	defer s.Stop()
	sendSegmentEvent(allClients, "Starting prep-node", auth, false)
	s.Suffix = " Starting prep-node"

	hostOS, err := ValidatePlatform(allClients.Executor)
	if err != nil {
		errStr := "Error: Invalid host OS. " + err.Error()
		sendSegmentEvent(allClients, errStr, auth, true)
		return fmt.Errorf(errStr)
	}

	if hostOS == "debian" {

		platform := debian.NewDebian(allClients.Executor)
		result, _ := platform.CheckIfdpkgISLock()
		if !result {
			zap.S().Error("Dpkg lock is acquired by another process while prep-node was running")
			return fmt.Errorf("Dpkg is under lock")
		}

		if ifUnattendedUpgradesAreEnabled(allClients) {
			DisableUnattendedUpgrades(allClients)
			defer EnableUnattendedUpgrades(allClients)
		} else {
			zap.S().Debug("unattended-upgrades are diabled by user")
		}
	}

	packagesPresent, newPackagesPresent, errStr := pf9PackagesPresent(hostOS, allClients.Executor, auth.Token, ctx.Fqdn)
	if errStr != nil {
		return fmt.Errorf("error while checking pf9 packages: %w", errStr)
	}
	//pf9ctl errors out if old packages are present
	if packagesPresent && (!newPackagesPresent || isCDU) {
		errStr := "\n\nOld Platform9 packages already present on the host." +
			"\nPlease uninstall these packages if you want to prep the node again.\n" +
			"Instructions to uninstall these are at:" +
			"\nhttps://docs.platform9.com/kubernetes/pmk-cli-unistall-hostagent"
		sendSegmentEvent(allClients, "Error: Platform9 packages already present.", auth, true)
		return fmt.Errorf(errStr)
	}

	//If new packages are not present, download and install them
	if !newPackagesPresent {
		sendSegmentEvent(allClients, "Installing hostagent - 2", auth, false)
		s.Suffix = " Downloading the Hostagent (this might take a few minutes...)"
		if err := installHostAgent(ctx, auth, hostOS, allClients.Executor); err != nil {
			errStr := "Error: Unable to install hostagent. " + err.Error()
			sendSegmentEvent(allClients, errStr, auth, true)
			return fmt.Errorf(errStr)
		}

		s.Suffix = " Platform9 packages installed successfully"

		if HostAgent == HostAgentCertless {
			s.Suffix = " Platform9 packages installed successfully"
			s.Stop()
			fmt.Println(color.Green("✓ ") + "Platform9 packages installed successfully")
		} else if HostAgent == HostAgentLegacy {
			s.Suffix = " Hostagent installed successfully"
			s.Stop()
			fmt.Println(color.Green("✓ ") + "Hostagent installed successfully")
		}
		s.Restart()
	}

	sendSegmentEvent(allClients, "Initialising host - 3", auth, false)
	s.Suffix = " Initialising host"
	zap.S().Debug("Initialising host")
	zap.S().Debug("Identifying the hostID from conf")
	cmd := `grep host_id /etc/pf9/host_id.conf | cut -d '=' -f2`
	output, err := allClients.Executor.RunWithStdout("bash", "-c", cmd)
	output = strings.TrimSpace(output)
	if err != nil || output == "" {
		errStr := "Error: Unable to fetch host ID. " + err.Error()
		sendSegmentEvent(allClients, errStr, auth, true)
		return fmt.Errorf(errStr)
	}

	s.Stop()
	fmt.Println(color.Green("✓ ") + "Initialised host successfully")
	zap.S().Debug("Initialised host successfully")
	if util.SkipKube {
		zap.S().Debug("Skip authorizing host as --skip-kube flag is true")
		sendSegmentEvent(allClients, "Successful", auth, false)
		return nil
	}

	s.Restart()
	s.Suffix = " Authorising host"
	zap.S().Debug("Authorising host")
	hostID := strings.TrimSuffix(output, "\n")
	time.Sleep(ctx.WaitPeriod * time.Second)

	if err := allClients.Resmgr.AuthorizeHost(hostID, auth.Token, util.KubeVersion); err != nil {
		errStr := "Error: Unable to authorise host. " + err.Error()
		sendSegmentEvent(allClients, errStr, auth, true)
		return fmt.Errorf(errStr)
	}

	zap.S().Debug("Host successfully attached to the Platform9 control-plane")
	s.Suffix = " Host successfully attached to the Platform9 control-plane"
	sendSegmentEvent(allClients, "Successful", auth, false)
	s.Stop()

	fmt.Println(color.Green("✓ ") + "Host successfully attached to the Platform9 control-plane")

	return nil
}

func ifUnattendedUpgradesAreEnabled(allClients client.Client) bool {
	zap.S().Debug("Checking if unattended-upgrades is enabled")
	output, err := allClients.Executor.RunWithStdout("bash", "-c", "cat /etc/apt/apt.conf.d/20auto-upgrades")
	if err != nil {
		zap.S().Debugf("Failed to check if unattended-upgrades are enabled : %s", err)
	}
	output = strings.TrimSpace(output)
	return !strings.Contains(output, `APT::Periodic::Unattended-Upgrade "0";`)
}

func DisableUnattendedUpgrades(allClients client.Client) {
	zap.S().Debug("Disabling unattended-upgrades")
	_, err := allClients.Executor.RunWithStdout("bash", "-c", "sed -i 's/[1-9]/0/g' /etc/apt/apt.conf.d/20auto-upgrades")
	if err != nil {
		zap.S().Debugf("Failed to disable unattended-upgrades : %s", err)
	} else {
		zap.S().Debug("Disabled unattended-upgrades")
	}
	_, err = allClients.Executor.RunWithStdout("bash", "-c", "systemctl restart unattended-upgrades")
	if err != nil {
		zap.S().Debug("Failed to restart unattended-upgrades service")
	}
}

func EnableUnattendedUpgrades(allClients client.Client) {
	zap.S().Debug("Enabling unattended-upgrades")
	_, err := allClients.Executor.RunWithStdout("bash", "-c", "sed -i 's/0/1/g' /etc/apt/apt.conf.d/20auto-upgrades")
	if err != nil {
		zap.S().Debugf("Failed to enable unattended-upgrades : %s", err)
	} else {
		zap.S().Debug("Enabled unattended-upgrades")
	}
	_, err = allClients.Executor.RunWithStdout("bash", "-c", "systemctl restart unattended-upgrades")
	if err != nil {
		zap.S().Debug("Failed to restart unattended-upgrades service")
	}
}

func installHostAgent(ctx objects.Config, auth keystone.KeystoneAuth, hostOS string, exec cmdexec.Executor) error {
	zap.S().Debug("Downloading the Hostagent (this might take a few minutes...)")

	regionURL, err := keystone.FetchRegionFQDN(ctx.Fqdn, ctx.Region, auth)
	if err != nil {
		return fmt.Errorf("Unable to fetch URL: %w", err)
	}

	url := fmt.Sprintf("https://%s/clarity/platform9-install-%s.sh", regionURL, hostOS)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("Unable to create a http request: %w", err)
	}

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Unable to send a request to clientL %w", err)
	}
	HostAgent = resp.StatusCode
	switch resp.StatusCode {
	case 404:
		return installHostAgentLegacy(ctx, regionURL, auth, hostOS, exec)
	case 200:
		return installHostAgentCertless(ctx, regionURL, auth, hostOS, exec)
	default:
		return fmt.Errorf("Invalid status code when identifiying hostagent type: %d", resp.StatusCode)
	}
}

func installHostAgentCertless(ctx objects.Config, regionURL string, auth keystone.KeystoneAuth, hostOS string, exec cmdexec.Executor) error {
	zap.S().Debug("Downloading the installer (this might take a few minutes...)")

	url := fmt.Sprintf(
		"https://%s/clarity/platform9-install-%s.sh",
		regionURL, hostOS)
	insecureDownload := ""
	if ctx.AllowInsecure {
		insecureDownload = "-k"
	}

	homeDir, err := createDirToDownloadInstaller(exec)
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf(`curl %s --silent --show-error  %s -o  %s/pf9/installer.sh`, insecureDownload, url, homeDir)
	_, err = exec.RunWithStdout("bash", "-c", cmd)
	if err != nil {
		return err
	}
	zap.S().Debug("Hostagent download completed successfully")

	var installOptions string

	//Pass keystone token if MFA token is provided
	if ctx.MfaToken != "" {
		installOptions = fmt.Sprintf(`--no-project --controller=%s  --user-token='%s'`, regionURL, auth.Token)
	} else {
		installOptions = fmt.Sprintf(`--no-project --controller=%s --username=%s --password='%s'`, regionURL, ctx.Username, ctx.Password)
	}
	if ctx.AllowInsecure {
		installOptions = fmt.Sprintf("%s --insecure", installOptions)
	}

	changePermission := fmt.Sprintf("chmod +x %s/pf9/installer.sh", homeDir)
	_, err = exec.RunWithStdout("bash", "-c", changePermission)
	if err != nil {
		return err
	}

	if ctx.ProxyURL != "" {
		cmd = fmt.Sprintf(`%s/pf9/installer.sh --proxy %s --skip-os-check --no-ntp`, homeDir, ctx.ProxyURL)
	} else {
		cmd = fmt.Sprintf(`%s/pf9/installer.sh --no-proxy --skip-os-check --no-ntp`, homeDir)
	}

	if IsRemoteExecutor {
		cmd = fmt.Sprintf(`bash %s %s`, cmd, installOptions)
		_, err = exec.RunWithStdout(cmd)
	} else {
		cmd = fmt.Sprintf(`%s %s`, cmd, installOptions)
		_, err = exec.RunWithStdout("bash", "-c", cmd)
	}

	removeTempDirAndInstaller(exec)

	if err != nil {
		_, exitCode := cmdexec.ExitCodeChecker(err)
		fmt.Printf("\n")
		fmt.Println("Error :", util.InstallerErrors[exitCode])
		zap.S().Debugf("Error:%s", util.InstallerErrors[exitCode])
		return fmt.Errorf("error while running installer script: %s", util.InstallerErrors[exitCode])
	}

	// TODO: here we actually need additional validation by checking /tmp/agent_install. log
	zap.S().Debug("Platform9 packages installed successfully")
	return nil
}

func removeTempDirAndInstaller(exec cmdexec.Executor) {
	zap.S().Debug("Removing temporary directory created to extract installer")
	removeTmpDirCmd := fmt.Sprintf("rm -rf %s/pf9/pf9-install-*", homeDir)
	_, err1 := exec.RunWithStdout("bash", "-c", removeTmpDirCmd)
	if err1 != nil {
		zap.S().Debug("error removing temporary directory")
	}

	zap.S().Debug("Removing installer script")
	removeInstallerCmd := fmt.Sprintf("rm -rf %s/pf9/installer.sh", homeDir)
	_, err1 = exec.RunWithStdout("bash", "-c", removeInstallerCmd)
	if err1 != nil {
		zap.S().Debug("error removing installer script")
	}

	zap.S().Debug("Removing legacy installer script")
	removeInstallerCmd = fmt.Sprintf("rm -rf %s/pf9/agent_install", homeDir)
	_, err1 = exec.RunWithStdout("bash", "-c", removeInstallerCmd)
	if err1 != nil {
		zap.S().Debug("error removing installer script")
	}
}

func ValidatePlatform(exec cmdexec.Executor) (string, error) {
	zap.S().Debug("Received a call to validate platform")

	strData, err := OpenOSReleaseFile(exec)
	if err != nil {
		return "", fmt.Errorf("failed reading data from file: %s", err)
	}
	var osplatform platform.Platform
	switch {
	case strings.Contains(strData, util.Centos) || strings.Contains(strData, util.Redhat) || strings.Contains(strData, util.Rocky):
		osplatform = centos.NewCentOS(exec)
		osVersion, err := osplatform.Version()
		if err == nil {
			return osVersion, nil
		} else if platform.SkipOSChecks && strings.Contains(err.Error(), "Unable to determine OS type") {
			zap.S().Info(err.Error())
			zap.S().Info("This OS version is not supported. Continuing as --skip-os-checks flag was used")
			return "redhat", nil
		} else {
			return "", fmt.Errorf("error in fetching OS version: %s", err.Error())
		}
	case strings.Contains(strData, util.Ubuntu):
		osplatform = debian.NewDebian(exec)
		osVersion, err := osplatform.Version()
		if err == nil {
			return osVersion, nil
		} else if platform.SkipOSChecks && strings.Contains(err.Error(), "Unable to determine OS type") {
			zap.S().Info(err.Error())
			zap.S().Info("This OS version is not supported. Continuing as --skip-os-checks flag was used")
			return "debian", nil
		} else {
			return "", fmt.Errorf("error in fetching OS version: %s", err.Error())
		}
	}

	return "", nil
}

func OpenOSReleaseFile(exec cmdexec.Executor) (string, error) {
	data, err := exec.RunWithStdout("cat", "/etc/os-release")
	if err != nil {
		return "", fmt.Errorf("failed reading data from file: %s", err)
	}
	return strings.ToLower(string(data)), nil
}

func pf9PackagesPresent(hostOS string, exec cmdexec.Executor, token string, fqdn string) (bool, bool, error) {
	var packagesPresent, newPackagesPresent bool = false, false
	var pkgCheckCommand, ext string
	pattern := `([^-\d]+)-(\d+\.\d+\.\d+-\d+)`
	reg := regexp.MustCompile(pattern)

	if hostOS == "debian" {
		pkgCheckCommand = "dpkg -l"
		ext = ".deb"
	} else {
		pkgCheckCommand = "yum list installed"
		ext = ".rpm"
	}

	for _, p := range util.Pf9Packages {
		cmd := fmt.Sprintf("%s | { grep -i '%s' || true; }", pkgCheckCommand, p)
		out, _ := exec.RunWithStdout("bash", "-c", cmd)
		if out != "" {
			packagesPresent = true
			break
		}
	}

	if !packagesPresent {
		zap.S().Infof("Pf9 packages are not present")
		return false, false, nil
	}
	//If pkgs are present, check version
	url := fmt.Sprintf("%s/protected/nocert-packagelist%s", fqdn, ext)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return true, false, fmt.Errorf("Unable to create a http request to list packages: %w", err)
	}
	req.Header.Set("X-Auth-Token", token)
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return true, false, fmt.Errorf("Unable to send a request to client %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return true, false, fmt.Errorf("error reading response body: %w", err)
		}
		lines := strings.Split(string(body), "\n")
		for _, line := range lines {
			if strings.Contains(line, ext) {
				match := reg.FindStringSubmatch(line)
				if len(match) <= 2 {
					return true, false, fmt.Errorf("error in extracting version from packages")
				}
				cmd := fmt.Sprintf("%s | { grep -i '%s.*%s' || true; }", pkgCheckCommand, match[1], match[2])
				out, _ := exec.RunWithStdout("bash", "-c", cmd)
				if out != "" {
					newPackagesPresent = true
					return packagesPresent, true, nil
				}
			}
		}
		return packagesPresent, false, nil
	}

	if resp.StatusCode == http.StatusFound {
		isCDU = true
		return true, false, nil
	} else {
		return true, false, fmt.Errorf("Error: List packages request returned status code: %d", resp.StatusCode)
	}

	return packagesPresent, newPackagesPresent, nil
}

func installHostAgentLegacy(ctx objects.Config, regionURL string, auth keystone.KeystoneAuth, hostOS string, exec cmdexec.Executor) error {
	zap.S().Debug("Downloading Hostagent Installer Legacy")

	url := fmt.Sprintf("https://%s/private/platform9-install-%s.sh", regionURL, hostOS)

	homeDir, err := createDirToDownloadInstaller(exec)
	if err != nil {
		return err
	}

	installOptions := fmt.Sprintf("--insecure --project-name=%s 2>&1 | tee -a %s/pf9/agent_install", auth.ProjectID, homeDir)
	//use insecure by default
	cmd := fmt.Sprintf(`curl --insecure --silent --show-error -H 'X-Auth-Token:%s' %s -o %s/pf9/installer.sh`, auth.Token, url, homeDir)
	_, err = exec.RunWithStdout("bash", "-c", cmd)
	if err != nil {
		return err
	}

	zap.S().Debug("Hostagent download completed successfully")
	changePermission := fmt.Sprintf("chmod +x %s/pf9/installer.sh", homeDir)
	_, err = exec.RunWithStdout("bash", "-c", changePermission)
	if err != nil {
		return err
	}

	if ctx.ProxyURL != "" {
		cmd = fmt.Sprintf(`%s/pf9/installer.sh --proxy %s --skip-os-check --no-ntp`, homeDir, ctx.ProxyURL)
	} else {
		cmd = fmt.Sprintf(`%s/pf9/installer.sh --no-proxy --skip-os-check --no-ntp`, homeDir)
	}

	if IsRemoteExecutor {
		cmd = fmt.Sprintf(`bash %s %s`, cmd, installOptions)
		_, err = exec.RunWithStdout(cmd)
	} else {
		cmd = fmt.Sprintf(`%s %s`, cmd, installOptions)
		_, err = exec.RunWithStdout("bash", "-c", cmd)
	}

	removeTempDirAndInstaller(exec)

	if err != nil {
		_, exitCode := cmdexec.ExitCodeChecker(err)
		fmt.Printf("\n")
		zap.S().Debugf("Error:%s", util.InstallerErrors[exitCode])
		fmt.Println("Error :", util.InstallerErrors[exitCode])
		return fmt.Errorf("error while running installer script: %s", util.InstallerErrors[exitCode])
	}

	// TODO: here we actually need additional validation by checking /tmp/agent_install. log
	zap.S().Debug("Hostagent installed successfully")
	return nil
}

func CheckSudo(exec cmdexec.Executor) bool {
	_, err := exec.RunWithStdout("-l")
	return err == nil
}

func createDirToDownloadInstaller(exec cmdexec.Executor) (string, error) {
	var err error
	homeDir, err = exec.RunWithStdout("bash", "-c", "echo $HOME")
	homeDir = strings.TrimSpace(strings.Trim(homeDir, "\n\""))
	if err != nil {
		return "", err
	}
	//creating this dir because in remote case this dir will not be present for fresh vm
	//for local case it will not cause any problem
	cmd := fmt.Sprintf(`%s/pf9`, homeDir)
	_, err = exec.RunWithStdout("mkdir", "-p", cmd)
	if err != nil {
		zap.S().Debugf("Directory exists")
	}
	return homeDir, nil
}
