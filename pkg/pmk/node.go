// Copyright © 2020 The Platform9 Systems Inc.
package pmk

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/keystone"
	"github.com/platform9/pf9ctl/pkg/platform"
	"github.com/platform9/pf9ctl/pkg/platform/centos"
	"github.com/platform9/pf9ctl/pkg/platform/debian"
	"github.com/platform9/pf9ctl/pkg/util"
	"go.uber.org/zap"
)

// This variable is assigned with StatusCode during hostagent installation
var HostAgent int
var IsRemoteExecutor bool

const (
	// Response Status Codes
	HostAgentCertless = 200
	HostAgentLegacy   = 404
)

// Sends an event to segment based on the input string and uses auth as keystone UUID property.
func sendSegmentEvent(allClients Client, eventStr string, auth keystone.KeystoneAuth, isError bool) {

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
		zap.S().Errorf("Unable to send Segment event for Node prep. Error: %s", err.Error())
	}

	// Close the segment for error path. Cmd level closure does not work due to FatalF.
	if isError {
		defer allClients.Segment.Close()
	}
}

// PrepNode sets up prerequisites for k8s stack
func PrepNode(ctx Config, allClients Client) error {
	// Building our new spinner
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Color("red")

	zap.S().Debug("Received a call to start preparing node(s).")

	auth, err := allClients.Keystone.GetAuth(
		ctx.Username,
		ctx.Password,
		ctx.Tenant,
	)

	if err != nil {
		return fmt.Errorf("Unable to locate keystone credentials: %s", err.Error())
	}
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

	present := pf9PackagesPresent(hostOS, allClients.Executor)
	if present {
		errStr := "\n\nPlatform9 packages already present on the host." +
			"\nPlease uninstall these packages if you want to prep the node again.\n" +
			"Instructions to uninstall these are at:" +
			"\nhttps://docs.platform9.com/kubernetes/pmk-cli-unistall-hostagent"
		sendSegmentEvent(allClients, "Error: Platform9 packages already present.", auth, true)
		return fmt.Errorf(errStr)
	}

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

	sendSegmentEvent(allClients, "Initialising host - 3", auth, false)
	s.Suffix = " Initialising host"
	zap.S().Debug("Initialising host")
	zap.S().Debug("Identifying the hostID from conf")
	cmd := `cat /etc/pf9/host_id.conf | grep ^host_id | cut -d = -f2 | cut -d ' ' -f2`
	output, err := allClients.Executor.RunWithStdout("bash", "-c", cmd)

	if err != nil {
		errStr := "Error: Unable to fetch host ID. " + err.Error()
		sendSegmentEvent(allClients, errStr, auth, true)
		return fmt.Errorf(errStr)
	}

	s.Stop()
	fmt.Println(color.Green("✓ ") + "Initialised host successfully")
	s.Restart()
	s.Suffix = " Authorising host"
	hostID := strings.TrimSuffix(output, "\n")
	time.Sleep(ctx.WaitPeriod * time.Second)

	sendSegmentEvent(allClients, "Authorising host - 4", auth, false)
	if err := allClients.Resmgr.AuthorizeHost(hostID, auth.Token); err != nil {
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

func FetchRegionFQDN(ctx Config, auth keystone.KeystoneAuth) (string, error) {

	// "regionInfo" service will have endpoint information. So fetch it's service ID.
	regionInfoServiceID, err := keystone.GetServiceID(ctx.Fqdn, auth, "regionInfo")
	if err != nil {
		return "", fmt.Errorf("Failed to fetch installer URL, Error: %s", err)
	}
	zap.S().Debug("Service ID fetched : ", regionInfoServiceID)

	// Fetch the endpoint based on region name.
	endpointURL, err := keystone.GetEndpointForRegion(ctx.Fqdn, auth, ctx.Region, regionInfoServiceID)
	if err != nil {
		return "", fmt.Errorf("Failed to fetch installer URL, Error: %s", err)
	}
	zap.S().Debug("endpointURL fetched : ", endpointURL)
	return endpointURL, nil
}

func installHostAgent(ctx Config, auth keystone.KeystoneAuth, hostOS string, exec cmdexec.Executor) error {
	zap.S().Debug("Downloading Hostagent")

	regionURL, err := FetchRegionFQDN(ctx, auth)
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

func installHostAgentCertless(ctx Config, regionURL string, auth keystone.KeystoneAuth, hostOS string, exec cmdexec.Executor) error {
	zap.S().Debug("Downloading the installer (this might take a few minutes...)")

	url := fmt.Sprintf(
		"https://%s/clarity/platform9-install-%s.sh",
		regionURL, hostOS)
	insecureDownload := ""
	if ctx.AllowInsecure {
		insecureDownload = "-k"
	}
	cmd := fmt.Sprintf(`curl %s --silent --show-error  %s -o  /tmp/installer.sh`, insecureDownload, url)
	_, err := exec.RunWithStdout("bash", "-c", cmd)
	if err != nil {
		return err
	}
	zap.S().Debug("Hostagent download completed successfully")

	installOptions := fmt.Sprintf(`--no-project --controller=%s --username=%s --password='%s'`, regionURL, ctx.Username, ctx.Password)

	_, err = exec.RunWithStdout("bash", "-c", "chmod +x /tmp/installer.sh")
	if err != nil {
		return err
	}

	if ctx.ProxyURL != "" {
		cmd = fmt.Sprintf(`/tmp/installer.sh --proxy http://%s --skip-os-check --ntpd %s`, ctx.ProxyURL, installOptions)
	} else {
		cmd = fmt.Sprintf(`/tmp/installer.sh --no-proxy --skip-os-check --ntpd %s`, installOptions)
	}

	if IsRemoteExecutor {
		cmd = fmt.Sprintf("bash %s", cmd)
		_, err = exec.RunWithStdout(cmd)
	} else {
		_, err = exec.RunWithStdout("bash", "-c", cmd)
	}
	if err != nil {
		return fmt.Errorf("Unable to run installer script")
	}

	// TODO: here we actually need additional validation by checking /tmp/agent_install. log
	zap.S().Debug("Platform9 packages installed successfully")
	return nil
}

func ValidatePlatform(exec cmdexec.Executor) (string, error) {
	zap.S().Debug("Received a call to validate platform")

	strData, err := openOSReleaseFile(exec)
	if err != nil {
		return "", fmt.Errorf("failed reading data from file: %s", err)
	}
	var platform platform.Platform
	switch {
	case strings.Contains(strData, util.Centos) || strings.Contains(strData, util.Redhat):
		platform = centos.NewCentOS(exec)
		osVersion, err := platform.Version()
		if err == nil {
			return osVersion, nil
		}
	case strings.Contains(strData, util.Ubuntu):
		platform = debian.NewDebian(exec)
		osVersion, err := platform.Version()
		if err == nil {
			return osVersion, nil
		}
	}

	return "", nil
}

func openOSReleaseFile(exec cmdexec.Executor) (string, error) {
	data, err := exec.RunWithStdout("cat", "/etc/os-release")
	if err != nil {
		return "", fmt.Errorf("failed reading data from file: %s", err)
	}
	return strings.ToLower(string(data)), nil
}

func pf9PackagesPresent(hostOS string, exec cmdexec.Executor) bool {
	var out string
	if hostOS == "debian" {
		out, _ = exec.RunWithStdout("bash",
			"-c",
			"dpkg -l | { grep -i 'pf9-' || true; }")
	} else {
		// not checking for redhat because if it has already passed validation
		// it must be either debian or redhat based
		out, _ = exec.RunWithStdout("bash",
			"-c",
			"yum list installed | { grep -i 'pf9-' || true; }")
	}

	return !(out == "")
}

func installHostAgentLegacy(ctx Config, regionURL string, auth keystone.KeystoneAuth, hostOS string, exec cmdexec.Executor) error {
	zap.S().Debug("Downloading Hostagent Installer Legacy")

	url := fmt.Sprintf("https://%s/private/platform9-install-%s.sh", regionURL, hostOS)
	installOptions := fmt.Sprintf("--insecure --project-name=%s 2>&1 | tee -a /tmp/agent_install", auth.ProjectID)
	//use insecure by default
	cmd := fmt.Sprintf(`curl --insecure --silent --show-error -H 'X-Auth-Token:%s' %s -o /tmp/installer.sh`, auth.Token, url)
	_, err := exec.RunWithStdout("bash", "-c", cmd)
	if err != nil {
		return err
	}

	zap.S().Debug("Hostagent download completed successfully")
	_, err = exec.RunWithStdout("bash", "-c", "chmod +x /tmp/installer.sh")
	if err != nil {
		return err
	}

	if ctx.ProxyURL != "" {
		cmd = fmt.Sprintf(`/tmp/installer.sh --proxy https://%s --skip-os-check --ntpd %s`, ctx.ProxyURL, installOptions)
	} else {
		cmd = fmt.Sprintf(`/tmp/installer.sh --no-proxy --skip-os-check --ntpd %s`, installOptions)
	}

	_, err = exec.RunWithStdout("bash", "-c", cmd)
	if err != nil {
		return err
	}

	// TODO: here we actually need additional validation by checking /tmp/agent_install. log
	zap.S().Debug("Hostagent installed successfully")
	return nil
}

func checkSudo(exec cmdexec.Executor) bool {
	_, err := exec.RunWithStdout("-l")
	return err == nil
}
