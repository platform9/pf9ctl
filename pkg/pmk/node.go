package pmk

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"runtime"
	"strings"

	"github.com/platform9/pf9ctl/pkg/log"
	"github.com/platform9/pf9ctl/pkg/pmk/clients"
	"github.com/platform9/pf9ctl/pkg/util"
)

// PrepNode sets up prerequisites for k8s stack
func PrepNode(
	ctx Context,
	c clients.Client,
	user string,
	password string,
	sshkey string,
	ips []string) error {

	log.Debug("Received a call to start preping node(s).")

	host, err := getHost("/etc/os-release", c.Executor)
	if err != nil {
		return fmt.Errorf("Invalid host os: %s", err.Error())
	}

	present := host.PackagePresent("pf9-")
	if present {
		return fmt.Errorf("Platform9 packages already present on the host. Please uninstall these packages if you want to prep the node again")
	}

	err = setupNode(host)
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

	if err := installHostAgent(ctx, auth, host.String()); err != nil {
		return fmt.Errorf("Unable to install hostagent: %w", err)
	}

	log.Debug("Identifying the hostID from conf")
	cmd := `cat /etc/pf9/host_id.conf | grep ^host_id | cut -d = -f2 | cut -d ' ' -f2`
	output, err := c.Executor.RunWithStdout("bash", "-c", cmd)

	if err != nil {
		return fmt.Errorf("Unable to fetch host ID for host authorization: %s", err.Error())
	}

	hostID := strings.TrimSuffix(output, "\n")

	if err := c.Resmgr.AuthorizeHost(hostID, auth.Token); err != nil {
		return err
	}

	if err := c.Segment.SendEvent("Prep Node - Successful", auth); err != nil {
		log.Errorf("Unable to send Segment event for Node prep. Error: %s", err.Error())
	}

	return nil
}

func installHostAgent(ctx Context, auth clients.KeystoneAuth, host string) error {
	log.Info("Download Hostagent")

	url := fmt.Sprintf("%s/clarity/platform9-install-%s.sh", ctx.Fqdn, host)
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
		return installHostAgentLegacy(ctx, auth, host)
	case 200:
		return installHostAgentCertless(ctx, auth, host)
	default:
		return fmt.Errorf("Invalid status code when identifiying hostagent type: %d", resp.StatusCode)
	}
}

func installHostAgentCertless(ctx Context, auth clients.KeystoneAuth, hostOS string) error {
	log.Info("Downloading Hostagent Installer Certless")

	url := fmt.Sprintf(
		"%s/clarity/platform9-install-%s.sh",
		ctx.Fqdn, hostOS)

	cmd := fmt.Sprintf(`curl --silent --show-error  %s -o  /tmp/installer.sh`, url)
	_, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return err
	}
	log.Debug("Hostagent download completed successfully")

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
	log.Info("Hostagent installed successfully")
	return nil
}

func getHost(hostReleaseLoc string, exec clients.Executor) (Host, error) {
	log.Debug("Received a call to validate platform")

	OS := runtime.GOOS
	if OS != "linux" {
		return nil, fmt.Errorf("Unsupported OS: %s", OS)
	}

	data, err := ioutil.ReadFile(hostReleaseLoc)
	if err != nil {
		return nil, fmt.Errorf("failed reading data from file: %w", err)
	}

	release := strings.ToLower(string(data))
	switch {
	case strings.Contains(release, "centos") || strings.Contains(release, "redhat"):
		out, err := exec.RunWithStdout(
			"bash",
			"-c",
			"cat /etc/*release | grep '(Core)' | grep 'CentOS Linux release' -m 1 | cut -f4 -d ' '")
		if err != nil {
			return nil, fmt.Errorf("Couldn't read the OS configuration file os-release: %w", err)
		}

		if util.StringContainsAny(out, []string{"7.5", "7.6", "7.7", "7.8"}) {
			return Redhat{exec}, nil
		}
		return nil, fmt.Errorf("Only %s versions of centos are supported", "7.5 7.6 7.7 7.8")

	case strings.Contains(release, "ubuntu"):
		out, err := exec.RunWithStdout(
			"bash",
			"-c",
			"cat /etc/*os-release | grep -i pretty_name | cut -d ' ' -f 2")
		if err != nil {
			return nil, fmt.Errorf("Couldn't read the OS configuration file os-release: %s", err.Error())
		}

		if util.StringContainsAny(out, []string{"16", "18"}) {
			return Debian{exec: exec}, nil
		}
		return nil, fmt.Errorf("Only %s versions of ubuntu are supported", "16 18")

	default:
		return nil, fmt.Errorf("Invalid release: %s", release)
	}
}

func installHostAgentLegacy(ctx Context, auth clients.KeystoneAuth, hostOS string) error {
	log.Info("Downloading Hostagent Installer Legacy")

	url := fmt.Sprintf("%s/private/platform9-install-%s.sh", ctx.Fqdn, hostOS)
	installOptions := fmt.Sprintf("--insecure --project-name=%s 2>&1 | tee -a /tmp/agent_install", auth.ProjectID)

	cmd := fmt.Sprintf(`curl --silent --show-error -H "X-Auth-Token: %s" %s -o /tmp/installer.sh`, auth.Token, url)
	_, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return err
	}

	log.Debug("Hostagent download completed successfully")
	_, err = exec.Command("bash", "-c", "chmod +x /tmp/installer.sh").Output()
	if err != nil {
		return err
	}

	if ctx.Proxy != "" {

	}

	cmd = fmt.Sprintf(`/tmp/installer.sh --no-proxy --skip-os-check --ntpd %s`, installOptions)
	_, err = exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return err
	}

	// TODO: here we actually need additional validation by checking /tmp/agent_install. log
	log.Info("Hostagent installed successfully")
	return nil
}
