package pmk

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/platform9/pf9ctl/pkg/constants"
	"github.com/platform9/pf9ctl/pkg/logger"
	"github.com/platform9/pf9ctl/pkg/pmk/clients"
)

// PrepNode sets up prerequisites for k8s stack
func PrepNode(
	ctx Context,
	c clients.Client,
	user string,
	password string,
	sshkey string,
	ips []string) error {

	logger.Log.Debug("Received a call to start preping node(s).")

	hostOS, err := validatePlatform()
	if err != nil {
		return fmt.Errorf("Invalid host os: %s", err.Error())
	}

	present := pf9PackagesPresent(hostOS, c.Executor)
	if present {
		return fmt.Errorf("Platform9 packages already present on the host. Please uninstall these packages if you want to prep the node again")
	}

	err = setupNode(hostOS)
	if err != nil {
		return fmt.Errorf("Unable to setup node: %s", err.Error())
	}

	auth, err := c.Keystone.GetAuth(
		ctx.Username,
		ctx.Password,
		ctx.Tenant,
	)

	if err != nil {
		return fmt.Errorf("Unable to locate keystone credentials: %s", err.Error())
	}

	if err := installHostAgent(ctx, auth, hostOS); err != nil {
		return fmt.Errorf("Unable to install hostagent: %w", err)
	}

	logger.Log.Debug("Identifying the hostID from conf")
	cmd := `cat /etc/pf9/host_id.conf | grep ^host_id | cut -d = -f2 | cut -d ' ' -f2`
	output, err := c.Executor.RunWithStdout("bash", "-c", cmd)

	if err != nil {
		return fmt.Errorf("Unable to fetch host ID for host authorization: %s", err.Error())
	}

	hostID := strings.TrimSuffix(output, "\n")
	time.Sleep(constants.WaitPeriod * time.Second)

	if err := c.Resmgr.AuthorizeHost(hostID, auth.Token); err != nil {
		return err
	}

	if err := c.Segment.SendEvent("Prep Node - Successful", auth); err != nil {
		logger.Log.Errorf("Unable to send Segment event for Node prep. Error: %s", err.Error())
	}

	return nil
}

func installHostAgent(ctx Context, auth clients.KeystoneAuth, hostOS string) error {
	logger.Log.Debug("Downloading Hostagent")

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
		return installHostAgentLegacy(ctx, auth, hostOS)
	case 200:
		return installHostAgentCertless(ctx, auth, hostOS)
	default:
		return fmt.Errorf("Invalid status code when identifiying hostagent type: %d", resp.StatusCode)
	}
}

func installHostAgentCertless(ctx Context, auth clients.KeystoneAuth, hostOS string) error {
	logger.Log.Info("Downloading Hostagent Installer Certless")

	url := fmt.Sprintf(
		"%s/clarity/platform9-install-%s.sh",
		ctx.Fqdn, hostOS)

	cmd := fmt.Sprintf(`curl --silent --show-error  %s -o  /tmp/installer.sh`, url)
	_, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return err
	}
	logger.Log.Debug("Hostagent download completed successfully")

	// Decoding base64 encoded password
	decodedBytePassword, err := base64.StdEncoding.DecodeString(ctx.Password)
	if err != nil {
		return err
	}
	decodedPassword := string(decodedBytePassword)
	installOptions := fmt.Sprintf(`--no-project --controller=%s --username=%s --password=%s`, ctx.Fqdn, ctx.Username, decodedPassword)

	_, err = exec.Command("bash", "-c", "chmod +x /tmp/installer.sh").Output()
	if err != nil {
		return err
	}

	cmd = fmt.Sprintf(`/tmp/installer.sh --no-proxy --skip-os-check --ntpd %s`, installOptions)
	_, err = exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return err
	}

	// TODO: here we actually need additional validation by checking /tmp/agent_install. log
	logger.Log.Info("Hostagent installed successfully")
	return nil
}

func validatePlatform() (string, error) {
	logger.Log.Debug("Received a call to validate platform")

	OS := runtime.GOOS
	if OS != "linux" {
		return "", fmt.Errorf("Unsupported OS: %s", OS)
	}

	data, err := ioutil.ReadFile("/etc/os-release")
	if err != nil {
		return "", fmt.Errorf("failed reading data from file: %s", err)
	}

	strDataLower := strings.ToLower(string(data))
	switch {
	case strings.Contains(strDataLower, "centos") || strings.Contains(strDataLower, "redhat"):
		out, err := exec.Command(
			"bash",
			"-c",
			"cat /etc/*release | grep '(Core)' | grep 'CentOS Linux release' -m 1 | cut -f4 -d ' '").Output()
		if err != nil {
			return "", fmt.Errorf("Couldn't read the OS configuration file os-release: %s", err.Error())
		}
		if strings.Contains(string(out), "7.5") || strings.Contains(string(out), "7.6") || strings.Contains(string(out), "7.7") || strings.Contains(string(out), "7.8") {
			return "redhat", nil
		}

	case strings.Contains(strDataLower, "ubuntu"):
		out, err := exec.Command(
			"bash",
			"-c",
			"cat /etc/*os-release | grep -i pretty_name | cut -d ' ' -f 2").Output()
		if err != nil {
			return "", fmt.Errorf("Couldn't read the OS configuration file os-release: %s", err.Error())
		}
		if strings.Contains(string(out), "16") || strings.Contains(string(out), "18") {
			return "debian", nil
		}
	}

	return "", nil
}

func pf9PackagesPresent(hostOS string, exec clients.Executor) bool {
	var err error
	if hostOS == "debian" {
		err = exec.Run("bash",
			"-c",
			"dpkg -l | grep -i 'pf9-'")
	} else {
		// not checking for redhat because if it has already passed validation
		// it must be either debian or redhat based
		err = exec.Run("bash",
			"-c",
			"yum list | grep -i 'pf9-'")
	}

	return err == nil
}

func installHostAgentLegacy(ctx Context, auth clients.KeystoneAuth, hostOS string) error {
	logger.Log.Info("Downloading Hostagent Installer Legacy")

	url := fmt.Sprintf("%s/private/platform9-install-%s.sh", ctx.Fqdn, hostOS)
	installOptions := fmt.Sprintf("--insecure --project-name=%s 2>&1 | tee -a /tmp/agent_install.log", auth.ProjectID)

	cmd := fmt.Sprintf(`curl --silent --show-error -H "X-Auth-Token: %s" %s -o /tmp/installer.sh`, auth.Token, url)
	_, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return err
	}

	logger.Log.Debug("Hostagent download completed successfully")
	_, err = exec.Command("bash", "-c", "chmod +x /tmp/installer.sh").Output()
	if err != nil {
		return err
	}

	cmd = fmt.Sprintf(`/tmp/installer.sh --no-proxy --skip-os-check --ntpd %s`, installOptions)
	_, err = exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return err
	}

	// TODO: here we actually need additional validation by checking /tmp/agent_install. log
	logger.Log.Info("Hostagent installed successfully")
	return nil
}
