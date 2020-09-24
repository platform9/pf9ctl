package pmk

import (
	"context"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	rhttp "github.com/hashicorp/go-retryablehttp"
	"github.com/platform9/pf9ctl/pkg/log"
)

var (

	// A regular expression to match the error returned by net/http when the
	// configured number of redirects is exhausted. This error isn't typed
	// specifically so we resort to matching on the error string.
	redirectsErrorRe = regexp.MustCompile(`stopped after \d+ redirects\z`)

	// A regular expression to match the error returned by net/http when the
	// scheme specified in the URL is invalid. This error isn't typed
	// specifically so we resort to matching on the error string.
	schemeErrorRe = regexp.MustCompile(`unsupported protocol scheme`)
)

// PrepNode sets up prerequisites for k8s stack
func PrepNode(
	ctx Context,
	user string,
	password string,
	sshkey string,
	ips []string) error {

	log.Info.Println("Received a call to start preping node(s).")

	info, err := os.Stat("/etc/pf9/host_id.conf")

	if info != nil {
		fmt.Println("Node is already prepped.")
		return err
	}

	hostOS, err := validatePlatform()
	if err != nil {
		return fmt.Errorf("Invalid host os: %s", err.Error())
	}

	c := `cat /etc/pf9/host_id.conf`
	_, err = exec.Command("bash", "-c", c).Output()

	err = setupNode(hostOS)
	if err != nil {
		return fmt.Errorf("Unable to setup node: %s", err.Error())
	}

	keystoneAuth, err := getKeystoneAuth(
		ctx.Fqdn,
		ctx.Username,
		ctx.Password,
		ctx.Tenant)

	if err != nil {
		return fmt.Errorf("Unable to locate keystone credentials: %s", err.Error())
	}

	if err := installHostAgent(ctx, keystoneAuth, hostOS); err != nil {
		return fmt.Errorf("Unable to install hostagent: %s", err.Error())
	}

	log.Info.Println("Identifying the hostID from conf")
	cmd := `cat /etc/pf9/host_id.conf | grep ^host_id | cut -d = -f2 | cut -d ' ' -f2`
	byt, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return fmt.Errorf("Unable to fetch host ID for host authorization: %s", err.Error())
	}

	hostID := strings.TrimSuffix(string(byt[:]), "\n")
	time.Sleep(WaitPeriod * time.Second)
	return authorizeHost(
		hostID,
		keystoneAuth.Token,
		ctx.Fqdn)
}

func installHostAgent(ctx Context, keystoneAuth KeystoneAuth, hostOS string) error {
	log.Info.Println("Downloading Hostagent installer Certless")

	hostagentInstaller := fmt.Sprintf(
		"%s/clarity/platform9-install-%s.sh",
		ctx.Fqdn, hostOS)

	cmd := fmt.Sprintf(`curl --silent --show-error  %s -o  /tmp/installer.sh`, hostagentInstaller)
	fmt.Println(cmd)
	_, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return err
	}
	log.Info.Println("Hostagent download completed successfully")

	cmd = fmt.Sprintf(`--no-project --controller=%s --username=%s --password=%s`, ctx.Fqdn, ctx.Username, ctx.Password)

	_, err = exec.Command("bash", "-c", "chmod +x /tmp/installer.sh").Output()
	if err != nil {
		return err
	}

	cmd = fmt.Sprintf(`sudo /tmp/installer.sh --no-proxy --skip-os-check --ntpd %s`, cmd)
	fmt.Println(cmd)
	_, err = exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return err
	}

	// TODO: here we actually need additional validation by checking /tmp/agent_install. log
	log.Info.Println("hostagent installed successfully")
	return nil
}

// retryPolicyOn404 is similar to the defaulRetryPolicy but
// which an additional check for 404 status.
func retryPolicyOn404(ctx context.Context, resp *http.Response, err error) (bool, error) {
	// do not retry on context.Canceled or context.DeadlineExceeded
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	if err != nil {
		if v, ok := err.(*url.Error); ok {
			// Don't retry if the error was due to too many redirects.
			if redirectsErrorRe.MatchString(v.Error()) {
				return false, nil
			}

			// Don't retry if the error was due to an invalid protocol scheme.
			if schemeErrorRe.MatchString(v.Error()) {
				return false, nil
			}

			// Don't retry if the error was due to TLS cert verification failure.
			if _, ok := v.Err.(x509.UnknownAuthorityError); ok {
				return false, nil
			}
		}

		// The error is likely recoverable so retry.
		return true, nil
	}

	// 429 Too Many Requests is recoverable. Sometimes the server puts
	// a Retry-After response header to indicate when the server is
	// available to start processing request from client.
	if resp.StatusCode == http.StatusTooManyRequests {
		return true, nil
	}

	// Check the response code. We retry on 500-range responses to allow
	// the server time to recover, as 500's are typically not permanent
	// errors and may relate to outages on the server side. This will catch
	// invalid response codes as well, like 0 and 999.
	if resp.StatusCode == 0 || resp.StatusCode == 404 || (resp.StatusCode >= 500 && resp.StatusCode != 501) {
		return true, nil
	}

	return false, nil
}

func authorizeHost(hostID, token, fqdn string) error {
	log.Info.Printf("Received a call to authorize host: %s to fqdn: %s\n", hostID, fqdn)

	client := rhttp.NewClient()
	client.RetryMax = 5
	client.CheckRetry = rhttp.CheckRetry(retryPolicyOn404)

	url := fmt.Sprintf("%s/resmgr/v1/hosts/%s/roles/pf9-kube", fqdn, hostID)
	req, err := rhttp.NewRequest("PUT", url, nil)
	if err != nil {
		return fmt.Errorf("Unable to create a new request: %w", err)
	}

	req.Header.Set("X-Auth-Token", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("Unable to authorize host, code: %d", resp.StatusCode)
	}

	return nil
}

func validatePlatform() (string, error) {
	log.Info.Println("Received a call to validate platform")

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
			"sudo cat /etc/*release | grep '(Core)' | grep 'CentOS Linux release' -m 1 | cut -f4 -d ' '").Output()
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
			"sudo cat /etc/*os-release | grep -i pretty_name | cut -d ' ' -f 2").Output()
		if err != nil {
			return "", fmt.Errorf("Couldn't read the OS configuration file os-release: %s", err.Error())
		}
		if strings.Contains(string(out), "16") || strings.Contains(string(out), "18") {
			return "debian", nil
		}
	}

	return "", nil
}

func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Error.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP
}
