// Copyright © 2020 The Platform9 Systems Inc.

package pmk

import (
	"fmt"
	"strings"
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
	PASS             CheckNodeResult = "pass"
	RequiredFail     CheckNodeResult = "requiredFail"
	OptionalFail     CheckNodeResult = "optionalFail"
	CleanInstallFail CheckNodeResult = "cleanInstallFail"
)

/* This flag is set true, to have warning "!" message,
when user passes --skip-checks and optional checks fails.
*/
var WarningOptionalChecks bool

// CheckNode checks the prerequisites for k8s stack
func CheckNode(ctx objects.Config, allClients client.Client, auth keystone.KeystoneAuth, nc objects.NodeConfig) (CheckNodeResult, error) {
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
		return RequiredFail, fmt.Errorf("This OS is not supported. Supported operating systems are: Ubuntu (18.04, 20.04), CentOS 7.[3-9], RHEL 7.[3-9] & RHEL 8.[5-6]")
	}

	if err = allClients.Segment.SendEvent("Starting CheckNode", auth, checkPass, ""); err != nil {
		zap.S().Debugf("Unable to send Segment event for check node. Error: %s", err.Error())
	}

	s.Start() // Start the spinner
	defer s.Stop()
	zap.S().Debug("Running pre-requisite checks and installing any missing OS packages")
	s.Suffix = " Running pre-requisite checks and installing any missing OS packages"
	checks := platform.Check()
	s.Stop()

	//We will print console if any missing os packages installed
	if debian.MissingPkgsInstalledDebian || centos.MissingPkgsInstalledCentos {
		fmt.Printf(color.Green("✓ ") + "Missing package(s) installed successfully\n")
	}

	mandatoryCheck := true
	optionalCheck := true
	cleanInstallCheck := true

	for _, check := range checks {
		if check.Result {
			segment_str := "CheckNode: " + check.Name
			if err := allClients.Segment.SendEvent(segment_str, auth, checkPass, ""); err != nil {
				zap.S().Debugf("Unable to send Segment event for check node. Error: %s", err.Error())
			}
			fmt.Printf(color.Green("✓ ")+"%s\n", check.Name)

		} else {
			segment_str := "CheckNode: " + check.Name
			if err := allClients.Segment.SendEvent(segment_str, auth, checkFail, check.UserErr); err != nil {
				zap.S().Debugf("Unable to send Segment event for check node. Error: %s", err.Error())
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

		if check.Name == "Existing Platform9 Packages Check" && !check.Result {
			cleanInstallCheck = false
		}
	}

	if err = allClients.Segment.SendEvent("CheckNode complete", auth, checkPass, ""); err != nil {
		zap.S().Debugf("Unable to send Segment event for check node. Error: %s", err.Error())
	}
	fmt.Printf("\n")
	if mandatoryCheck {
		fmt.Println(color.Green("✓ ") + "Completed Pre-Requisite Checks successfully\n")
		zap.S().Debug("Completed Pre-Requisite Checks successfully")
	}

	removeCurrentInstallation := ""
	if !cleanInstallCheck {
		fmt.Println(color.Yellow("\nPrevious installation found"))
		if !nc.RemoveExistingPkgs {
			fmt.Println(color.Yellow("Reinstall Required..."))
			fmt.Print("Remove Current Installation Type ('yes'/'no'):")
			fmt.Scanf("%s", &removeCurrentInstallation)
		}
		if nc.RemoveExistingPkgs || strings.ToLower(removeCurrentInstallation) == "yes" {
			DecommissionNode(&ctx, nc, false)
			return CleanInstallFail, nil
		}

		return RequiredFail, nil

	}

	if !mandatoryCheck {
		return RequiredFail, nil
	} else if !optionalCheck {
		return OptionalFail, nil
	} else {
		return PASS, nil
	}

}
