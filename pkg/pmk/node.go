package pmk

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// PrepNode sets up prerequisites for k8s stack
func PrepNode(
	ctx Context,
	user string,
	password string,
	sshkey string,
	ips []string) error {

	log.Println("Received a call to start preping node(s).")

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

	// TODO: Common Functionality

	if err := installHostAgent(ctx, keystoneAuth, hostOS); err != nil {
		return fmt.Errorf("Unable to install hostagent: %s", err.Error())
	}

	log.Println("Identifying the hostID from conf")
	cmd := `cat /etc/pf9/host_id.conf | grep ^host_id | cut -d = -f2 | cut -d ' ' -f2`
	byt, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return fmt.Errorf("Unable to fetch host ID for host authorization: %s", err.Error())
	}

	hostID := strings.TrimSuffix(string(byt[:]), "\n")
	time.Sleep(60 * time.Second)
	return authorizeHost(
		hostID,
		keystoneAuth.Token,
		ctx.Fqdn)
}

func installHostAgent(ctx Context, keystoneAuth KeystoneAuth, hostOS string) error {
	log.Println("Downloading Hostagent installer")

	hostagentInstallOptions := fmt.Sprintf(
		"--insecure --project-name=%s 2>&1 > /tmp/agent_install.log",
		ctx.Tenant)

	hostagentInstaller := fmt.Sprintf(
		"%s/private/platform9-install-%s.sh",
		ctx.Fqdn, hostOS)

	cmd := fmt.Sprintf(`curl --silent --show-error -O -H "X-Auth-Token: %s" extra_opts="%s" %s > /tmp/installer.sh`,
		keystoneAuth.Token,
		hostagentInstallOptions,
		hostagentInstaller)
	fmt.Println(cmd)

	_, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return err
	}
	log.Println("Hostagent download completed successfully")
	_, err = exec.Command("bash", "-c", "chmod +x /tmp/installer.sh").Output()
	if err != nil {
		return err
	}

	cmd = fmt.Sprintf(`sudo /tmp/installer.sh --no-proxy --skip-os-check --ntpd %s`, hostagentInstallOptions)
	_, err = exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return err
	}

	// TODO: here we actually need additional validation by checking /tmp/agent_install. log
	log.Println("hostagent installed successfully")
	return nil
}

func authorizeHost(hostID, token, fqdn string) error {
	log.Printf("Received a call to authorize host: %s to fqdn: %s\n", hostID, fqdn)

	client := http.Client{}

	url := fmt.Sprintf("%s/resmgr/v1/hosts/%s/roles/pf9-kube", fqdn, hostID)
	fmt.Println(url)
	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	req.Header.Set("X-Auth-Token", token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("unable to add a new host to resmgr: %d", resp.StatusCode)
	}
	fmt.Println("Node added to resmgr successfully")

	return nil
}

func validatePlatform() (string, error) {
	log.Println("Received a call to validate platform")

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
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}
