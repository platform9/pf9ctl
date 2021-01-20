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
func CheckNode(allClients Client) bool {

	zap.S().Debug("Received a call to check node.")

	os, err := validatePlatform(allClients.Executor)
	if err != nil {
		return false
	}

	var platform platform.Platform
	switch os {
	case "debian":
		platform = debian.NewDebian(allClients.Executor)
	case "redhat":
		platform = centos.NewCentOS(allClients.Executor)
	}

	checks := platform.Check()
	result := true
	for _, check := range checks {
		if check.Result {
			fmt.Printf("%s : %s\n", check.Name, checkPass)
		} else {
			fmt.Printf("%s : %s\n", check.Name, checkFail)
			result = false
		}

		if check.Err != nil {
			zap.S().Debugf("Error in %s : %s", check.Name, err)
			result = false
		}
	}

	return result
}
