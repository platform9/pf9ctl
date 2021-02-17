// Copyright © 2020 The Platform9 Systems Inc.

package pmk

import (
	"fmt"

	"github.com/platform9/pf9ctl/pkg/platform"
	"github.com/platform9/pf9ctl/pkg/platform/centos"
	"github.com/platform9/pf9ctl/pkg/platform/debian"
	"go.uber.org/zap"
)

const (
	checkPass = "PASS"
	checkFail = "FAIL"
)

// CheckNode checks the prerequisites for k8s stack
func CheckNode(ctx Config, allClients Client) (bool, error) {

	zap.S().Debug("Received a call to check node.")

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
		return false, fmt.Errorf("Unable to locate keystone credentials: %s", err.Error())
	}

	checks := platform.Check()
	result := true
	for _, check := range checks {
		if check.Result {
			segment_str := "Check Node: " + check.Name + " Status: " + checkPass
			if err := allClients.Segment.SendEvent(segment_str, auth); err != nil {
				zap.S().Errorf("Unable to send Segment event for check node. Error: %s", err.Error())
			}
			fmt.Printf("%s : %s\n", check.Name, checkPass)
		} else {
			segment_str := "Check Node: " + check.Name + " Status: " + checkFail
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
