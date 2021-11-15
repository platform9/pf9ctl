// Copyright © 2020 The Platform9 Systems Inc.

package pmk

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/platform9/pf9ctl/pkg/client"
	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/keystone"
	"github.com/platform9/pf9ctl/pkg/objects"
	"github.com/platform9/pf9ctl/pkg/platform"
	"github.com/platform9/pf9ctl/pkg/platform/centos"
	"github.com/platform9/pf9ctl/pkg/platform/debian"
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

/* This flag is set true, to have warning "!" message,
when user passes --skipChecks and optional checks fails.
*/
var WarningOptionalChecks bool

// CheckNode checks the prerequisites for k8s stack

func CheckNode(ctx objects.Config, allClients client.Client, auth keystone.KeystoneAuth) (CheckNodeResult, error) {

	// Building our new spinner
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Color("red")

	zap.S().Debug("Received a call to check node.")

	isSudo := CheckSudo(allClients.Executor)
	if !isSudo {
		return RequiredFail, fmt.Errorf("User executing this CLI is not allowed to switch to privileged (sudo) mode")
	}
	os, err := ValidatePlatform(allClients.Executor)
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
		return RequiredFail, fmt.Errorf("This OS is not supported. Supported operating systems are: Ubuntu (16.04, 18.04, 20.04), CentOS 7.x & RHEL 7.x")
	}

	if err = allClients.Segment.SendEvent("Starting CheckNode", auth, checkPass, ""); err != nil {
		zap.S().Errorf("Unable to send Segment event for check node. Error: %s", err.Error())
	}

	s.Start() // Start the spinner
	defer s.Stop()
	s.Suffix = " Running pre-requisite checks and installing any missing OS packages"
	checks := platform.Check()
	s.Stop()

	//We will print console if any missing os packages installed
	if debian.MissingPkgsInstalledDebian || centos.MissingPkgsInstalledCentos {
		fmt.Printf(color.Green("✓ ") + "Missing package(s) installed successfully\n")
	}

	mandatoryCheck := true
	optionalCheck := true

	for _, check := range checks {
		if check.Result {
			segment_str := "CheckNode: " + check.Name
			if err := allClients.Segment.SendEvent(segment_str, auth, checkPass, ""); err != nil {
				zap.S().Errorf("Unable to send Segment event for check node. Error: %s", err.Error())
			}
			fmt.Printf(color.Green("✓ ")+"%s\n", check.Name)

		} else {
			segment_str := "CheckNode: " + check.Name
			if err := allClients.Segment.SendEvent(segment_str, auth, checkFail, check.UserErr); err != nil {
				zap.S().Errorf("Unable to send Segment event for check node. Error: %s", err.Error())
			}
			// To print warning "!", if --skipchecks flag passed and optional checks failed.
			if WarningOptionalChecks && !check.Mandatory {
				fmt.Printf(color.Yellow("! ")+"%s - %s\n", check.Name, check.UserErr)
			} else {
				fmt.Printf(color.Red("x ")+"%s - %s\n", check.Name, check.UserErr)
			}

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
	fmt.Printf("\n")
	if mandatoryCheck {
		fmt.Println(color.Green("✓ ") + "Completed Pre-Requisite Checks successfully\n")
	}

	if !mandatoryCheck {
		return RequiredFail, nil
	} else if !optionalCheck {
		return OptionalFail, nil
	} else {
		return PASS, nil
	}

}
