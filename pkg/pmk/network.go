package pmk

import (
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"go.uber.org/zap"
)

func HostId(exec cmdexec.Executor, fqdn string, token string, IPs []string) []string {
	zap.S().Debug("Getting host IDs")
	var hostIdsList []string
	tkn := fmt.Sprintf(`"X-Auth-Token: %v"`, token)
	for _, ip := range IPs {
		ip = fmt.Sprintf(`"%v"`, ip)
		cmd := fmt.Sprintf("curl -sH %v -X GET %v/resmgr/v1/hosts | jq -r '.[] | select(.extensions!=\"\")  | select(.extensions.ip_address.data[]==(%v)) | .id' ", tkn, fqdn, ip)
		hostid, _ := exec.RunWithStdout("bash", "-c", cmd)
		hostid = strings.TrimSpace(strings.Trim(hostid, "\n"))
		if len(hostid) == 0 {
			zap.S().Infof("Unable to find host with IP %v please try again or run prep-node first", ip)
		} else {
			hostIdsList = append(hostIdsList, hostid)
		}
	}
	return hostIdsList
}

func GetIp() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP
}
