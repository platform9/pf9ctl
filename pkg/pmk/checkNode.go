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

type CheckNodeResult string

const (
	PASS         CheckNodeResult = "pass"
	RequiredFail CheckNodeResult = "requiredFail"
	OptionalFail CheckNodeResult = "optionalFail"
)

// CheckNode checks the prerequisites for k8s stack
func CheckNode(ctx Config, allClients Client) (CheckNodeResult, error) {

	zap.S().Debug("Received a call to check node.")

	isSudo := checkSudo(allClients.Executor)
	if !isSudo {
		return RequiredFail, fmt.Errorf("User executing this CLI is not allowed to switch to privileged (sudo) mode")
	}

	os, err := validatePlatform(allClients.Executor)
	if err != nil {
		return RequiredFail, err
	}

	var platform platform.Platform
	switch os {
	case "debian":
		platform = debian.NewDebian(allClients.Executor)
	case "redhat":
		platform = centos.NewCentOS(allClients.Executor)
	default:
		return RequiredFail, fmt.Errorf("This OS is not supported. Supported operating systems are: Ubuntu (16.04, 18.04) & CentOS 7.x")
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

			return RequiredFail, fmt.Errorf("Possible clock skew detected. Check the system time and retry.")
		}
		return RequiredFail, fmt.Errorf("Unable to obtain keystone credentials: %s", err.Error())
	}

	if err = allClients.Segment.SendEvent("Starting CheckNode", auth, checkPass, ""); err != nil {
		zap.S().Errorf("Unable to send Segment event for check node. Error: %s", err.Error())
	}

	checks := platform.Check()
	mandatoryCheck := true
	optionalCheck := true

	fmt.Printf("\n\n")
	for _, check := range checks {
		if check.Result {
			segment_str := "CheckNode: " + check.Name
			if err := allClients.Segment.SendEvent(segment_str, auth, checkPass, ""); err != nil {
				zap.S().Errorf("Unable to send Segment event for check node. Error: %s", err.Error())
			}
			fmt.Printf("%s : %s\n", check.Name, checkPass)
		} else {
			segment_str := "CheckNode: " + check.Name
			if err := allClients.Segment.SendEvent(segment_str, auth, checkFail, check.UserErr); err != nil {
				zap.S().Errorf("Unable to send Segment event for check node. Error: %s", err.Error())
			}
			fmt.Printf("%s : %s - %s\n", check.Name, checkFail, check.UserErr)
			if check.Mandatory {
				mandatoryCheck = false
			} else {
				optionalCheck = false
			}
		}

		if check.Err != nil {
			zap.S().Debugf("Error in %s : %s", check.Name, check.Err)
		}
	}

	if err = allClients.Segment.SendEvent("CheckNode complete", auth, checkPass, ""); err != nil {
		zap.S().Errorf("Unable to send Segment event for check node. Error: %s", err.Error())
	}

	if !mandatoryCheck {
		return RequiredFail, nil
	} else if !optionalCheck {
		return OptionalFail, nil
	} else {
		return PASS, nil
	}

}
