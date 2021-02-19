// Copyright © 2020 The Platform9 Systems Inc.
package pmk

import (
	"fmt"
	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/keystone"
	"github.com/platform9/pf9ctl/pkg/platform"
	"github.com/platform9/pf9ctl/pkg/platform/centos"
	"github.com/platform9/pf9ctl/pkg/platform/debian"
	"github.com/platform9/pf9ctl/pkg/util"
	"go.uber.org/zap"
	"net/http"
	"strings"
	"time"
)

// Sends an event to segment based on the input string and uses auth as keystone UUID property.
func sendSegmentEvent(allClients Client, eventStr string, auth keystone.KeystoneAuth) {
	if err := allClients.Segment.SendEvent("Prep-node : "+eventStr, auth); err != nil {
		zap.S().Errorf("Unable to send Segment event for Node prep. Error: %s", err.Error())
	}
	// Segment events get posted from it's queue only after closing the client.
	allClients.Segment.Close()
}

// PrepNode sets up prerequisites for k8s stack
func PrepNode(ctx Config, allClients Client) error {

	zap.S().Debug("Received a call to start preping node(s).")

	auth, err := allClients.Keystone.GetAuth(
		ctx.Username,
		ctx.Password,
		ctx.Tenant,
	)

	if err != nil {
		return fmt.Errorf("Unable to locate keystone credentials: %s", err.Error())
	}

	hostOS, err := validatePlatform(allClients.Executor)
	if err != nil {
		errStr := "Error: Invalid host OS. " + err.Error()
		sendSegmentEvent(allClients, errStr, auth)
		return fmt.Errorf(errStr)
	}

	present := pf9PackagesPresent(hostOS, allClients.Executor)
	if present {
		errStr := "\n\nPlatform9 packages already present on the host." +
			"\nPlease uninstall these packages if you want to prep the node again.\n" +
			"Instructions to uninstall these are at:" +
			"\nhttps://docs.platform9.com/kubernetes/pmk-cli-unistall-hostagent"
		sendSegmentEvent(allClients, "Error: Platform9 packages already present.", auth)
		return fmt.Errorf(errStr)
	}

	err = setupNode(hostOS, allClients.Executor)
	if err != nil {
		errStr := "Error: Unable to disable swap. " + err.Error()
		sendSegmentEvent(allClients, errStr, auth)
		return fmt.Errorf(errStr)
	}

	if err := installHostAgent(ctx, auth, hostOS, allClients.Executor); err != nil {
		errStr := "Error: Unable to install hostagent. " + err.Error()
		sendSegmentEvent(allClients, errStr, auth)
		return fmt.Errorf(errStr)
	}

	zap.S().Debug("Identifying the hostID from conf")
	cmd := `cat /etc/pf9/host_id.conf | grep ^host_id | cut -d = -f2 | cut -d ' ' -f2`
	output, err := allClients.Executor.RunWithStdout("bash", "-c", cmd)

	if err != nil {
		errStr := "Error: Unable to fetch host ID. " + err.Error()
		sendSegmentEvent(allClients, errStr, auth)
		return fmt.Errorf(errStr)
	}

	hostID := strings.TrimSuffix(output, "\n")
	time.Sleep(ctx.WaitPeriod * time.Second)

	if err := allClients.Resmgr.AuthorizeHost(hostID, auth.Token); err != nil {
		errStr := "Error: Unable to authorize host. " + err.Error()
		sendSegmentEvent(allClients, errStr, auth)
		return fmt.Errorf(errStr)
	}

	sendSegmentEvent(allClients, "Successful", auth)

	return nil
}

func installHostAgent(ctx Config, auth keystone.KeystoneAuth, hostOS string, exec cmdexec.Executor) error {
	zap.S().Debug("Downloading Hostagent")

	url := fmt.Sprintf("%s/clarity/platform9-install-%s.sh", ctx.Fqdn, hostOS)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("Unable to create a http request: %w", err)
	}

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Unable to send a request to clientL %w", err)
	}

	switch resp.StatusCode {
	case 404:
		return installHostAgentLegacy(ctx, auth, hostOS, exec)
	case 200:
		return installHostAgentCertless(ctx, auth, hostOS, exec)
	default:
		return fmt.Errorf("Invalid status code when identifiying hostagent type: %d", resp.StatusCode)
	}
}

func installHostAgentCertless(ctx Config, auth keystone.KeystoneAuth, hostOS string, exec cmdexec.Executor) error {
	zap.S().Info("Downloading Hostagent Installer Certless")

	url := fmt.Sprintf(
		"%s/clarity/platform9-install-%s.sh",
		ctx.Fqdn, hostOS)
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

	installOptions := fmt.Sprintf(`--no-project --controller=%s --username=%s --password=%s`, ctx.Fqdn, ctx.Username, ctx.Password)

	_, err = exec.RunWithStdout("bash", "-c", "chmod +x /tmp/installer.sh")
	if err != nil {
		return err
	}

	cmd = fmt.Sprintf(`/tmp/installer.sh --no-proxy --skip-os-check --ntpd %s`, installOptions)
	_, err = exec.RunWithStdout("bash", "-c", cmd)
	if err != nil {
		return err
	}

	// TODO: here we actually need additional validation by checking /tmp/agent_install. log
	zap.S().Info("Hostagent installed successfully")
	return nil
}

func validatePlatform(exec cmdexec.Executor) (string, error) {
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

	fmt.Println(">>", out)

	return !(out == "")
}

func installHostAgentLegacy(ctx Config, auth keystone.KeystoneAuth, hostOS string, exec cmdexec.Executor) error {
	zap.S().Info("Downloading Hostagent Installer Legacy")

	url := fmt.Sprintf("%s/private/platform9-install-%s.sh", ctx.Fqdn, hostOS)
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

	cmd = fmt.Sprintf(`/tmp/installer.sh --no-proxy --skip-os-check --ntpd %s`, installOptions)
	_, err = exec.RunWithStdout("bash", "-c", cmd)
	if err != nil {
		return err
	}

	// TODO: here we actually need additional validation by checking /tmp/agent_install. log
	zap.S().Info("Hostagent installed successfully")
	return nil
}
