// Copyright © 2020 The Platform9 Systems Inc.

package pmk

import (
	"fmt"
	"strings"

	"github.com/platform9/pf9ctl/pkg/platform"
	"github.com/platform9/pf9ctl/pkg/platform/centos"
	"github.com/platform9/pf9ctl/pkg/platform/debian"
	"github.com/platform9/pf9ctl/pkg/util"
	"go.uber.org/zap"
)

const (
	checkPass = "PASS"
	checkFail = "FAIL"
)

// CheckNode checks the prerequisites for k8s stack
func CheckNode(ctx Config, allClients Client) (bool, error) {

	zap.S().Debug("Received a call to check node.")

	isSudo := checkSudo(allClients.Executor)
	if !isSudo {
		return false, fmt.Errorf("User executing this CLI is not allowed to switch to privileged (sudo) mode")
	}

	os, err := validatePlatform(allClients.Executor)
	if err != nil {
		return false, err
	}

	var platform platform.Platform
	switch os {
	case "debian":
		platform = debian.NewDebian(allClients.Executor)
	case "redhat":
		platform = centos.NewCentOS(allClients.Executor)
	}

	// Fetch the keystone token.
	// This is used as a reference to the segment event.
	auth, err := allClients.Keystone.GetAuth(
		ctx.Username,
		ctx.Password,
		ctx.Tenant,
	)

	if err != nil {
		// Certificate expiration is detected by the http library and
		// only error object gets populated, which means that the http
		// status code does not reflect the actual error code.
		// So parsing the err to check for certificate expiration.
		if strings.Contains(strings.ToLower(err.Error()), util.CertsExpireErr) {

			return false, fmt.Errorf("Possible clock skew detected. Check the system time and retry.")
		}
		return false, fmt.Errorf("Unable to obtain keystone credentials: %s", err.Error())
	}

	checks := platform.Check()
	result := true

	fmt.Printf("\n\n")
	for _, check := range checks {
		if check.Result {
			segment_str := "CheckNode: " + check.Name + " Status: " + checkPass
			if err := allClients.Segment.SendEvent(segment_str, auth); err != nil {
				zap.S().Errorf("Unable to send Segment event for check node. Error: %s", err.Error())
			}
			fmt.Printf("%s : %s\n", check.Name, checkPass)
		} else {
			segment_str := "CheckNode: " + check.Name + " Status: " + checkFail
			if err := allClients.Segment.SendEvent(segment_str, auth); err != nil {
				zap.S().Errorf("Unable to send Segment event for check node. Error: %s", err.Error())
			}
			fmt.Printf("%s : %s\n", check.Name, checkFail)
			result = false
		}

		if check.Err != nil {
			zap.S().Debugf("Error in %s : %s", check.Name, check.Err)
			result = false
		}
	}

	// Segment events get posted from it's queue only after closing the client.
	allClients.Segment.Close()
	return result, nil
}
