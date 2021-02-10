// Copyright © 2020 The Platform9 Systems Inc.
package pmk

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/keystone"
	"go.uber.org/zap"
)

// PrepNode sets up prerequisites for k8s stack
func PrepNode(ctx Config, allClients Client) error {

	zap.S().Debug("Received a call to start preping node(s).")

	hostOS, err := validatePlatform(allClients.Executor)
	if err != nil {
		return fmt.Errorf("Invalid host os: %s", err.Error())
	}

	present := pf9PackagesPresent(hostOS, allClients.Executor)
	if present {
		return fmt.Errorf("Platform9 packages already present on the host. Please uninstall these packages if you want to prep the node again")
	}

	err = setupNode(hostOS, allClients.Executor)
	if err != nil {
		return fmt.Errorf("Unable to setup node: %s", err.Error())
	}

	auth, err := allClients.Keystone.GetAuth(
		ctx.Username,
		ctx.Password,
		ctx.Tenant,
	)

	if err != nil {
		return fmt.Errorf("Unable to locate keystone credentials: %s", err.Error())
	}

	if err := installHostAgent(ctx, auth, hostOS, allClients.Executor); err != nil {
		return fmt.Errorf("Unable to install hostagent: %w", err)
	}

	zap.S().Debug("Identifying the hostID from conf")
	cmd := `cat /etc/pf9/host_id.conf | grep ^host_id | cut -d = -f2 | cut -d ' ' -f2`
	output, err := allClients.Executor.RunWithStdout("bash", "-c", cmd)

	if err != nil {
		return fmt.Errorf("Unable to fetch host ID for host authorization: %s", err.Error())
	}

	hostID := strings.TrimSuffix(output, "\n")
	time.Sleep(ctx.WaitPeriod * time.Second)

	if err := allClients.Resmgr.AuthorizeHost(hostID, auth.Token); err != nil {
		return err
	}

	if err := allClients.Segment.SendEvent("Prep Node - Successful", auth); err != nil {
		zap.S().Errorf("Unable to send Segment event for Node prep. Error: %s", err.Error())
	}

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
	
	strDataLower , err := openOSReleaseFile(exec)
	if err != nil {
		return "", fmt.Errorf("failed reading data from file: %s", err)
	}

	
	switch {
	case strings.Contains(strDataLower, "centos") || strings.Contains(strDataLower, "redhat"):
		osVersion , err := centosVersion(exec)
		if err == nil{
			return osVersion,nil
		}
	case strings.Contains(strDataLower, "ubuntu"):
		osVersion , err := ubuntuVersion(exec)
		if err == nil{
			return osVersion,nil
		}
	}

	return "", nil
}

func openOSReleaseFile(exec cmdexec.Executor) (string,error) {
	data, err := exec.RunWithStdout("cat", "/etc/os-release")
	if err != nil {
		return "", fmt.Errorf("failed reading data from file: %s", err)
	}
	return strings.ToLower(string(data)),nil
}

func centosVersion(exec cmdexec.Executor) (string, error) {
	out, err := exec.RunWithStdout(
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

func ubuntuVersion(exec cmdexec.Executor) (string, error) {
	out, err := exec.RunWithStdout(
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
