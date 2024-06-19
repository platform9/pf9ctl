// Copyright © 2020 The Platform9 Systems Inc.

package pmk

import (
	"fmt"
	"os"
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
	"github.com/platform9/pf9ctl/pkg/util"
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

/*
	This flag is set true, to have warning "!" message,

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
	hostOS, err := ValidatePlatform(allClients.Executor)
	if err != nil {
		return RequiredFail, err
	}

	var platform platform.Platform
	switch hostOS {
	case "debian":
		platform = debian.NewDebian(allClients.Executor)
	case "redhat":
		platform = centos.NewCentOS(allClients.Executor)
	case "debianOther":
		zap.S().Info("This OS version is not supported. Continuing as --skip-os-checks flag was used")
		platform = debian.NewDebian(allClients.Executor)
	case "redhatOther":
		zap.S().Info("This OS version is not supported. Continuing as --skip-os-checks flag was used")
		platform = centos.NewCentOS(allClients.Executor)
	default:
		return RequiredFail, fmt.Errorf("This OS is not supported. Supported operating systems are: Ubuntu (18.04, 20.04, 22.04), CentOS 7.[3-9], RHEL 7.[3-9] & RHEL 8.[5-9] & Rocky 9.[1-2]")
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

	if util.CheckIfOnboarded {
		zap.S().Debug("Checking if node is already connected to DU")
		for _, v := range checks {
			if v.Name == "Existing Platform9 Packages Check" {
				if !v.Result {
					zap.S().Debug("Platform9 Packages are present")
					zap.S().Debug("Checking if host is connected or not")
					// Directly use host_id instead of relying on IP to get host details
					// host_id file is created as part hostagent installation. File missing should mean
					// that installation was partial
					cmd := `grep host_id /etc/pf9/host_id.conf | cut -d '=' -f2`
					hostID, err := allClients.Executor.RunWithStdout("bash", "-c", cmd)
					if err != nil {
						zap.S().Debugf("Unable to get host id %s", err.Error())
					}
					hostID = strings.TrimSpace(hostID)
					connected := false
					if len(hostID) != 0 {
						connected = allClients.Resmgr.HostStatus(auth.Token, hostID)
					}
					if connected {
						zap.S().Debug("Node is already connected")
						zap.S().Info(color.Green("✓ ") + "Node is already connected\n")
						os.Exit(0)
					} else {
						//case where hostagent is installed but host is in disconnected sate
						zap.S().Debug("Hostagent is installed but host is not connected to DU. Installing hostagent again")
						nc.RemoveExistingPkgs = true
					}
				} else {
					zap.S().Debug("Node is not connected installing hostagent")
				}
			}
		}
	}
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
		if !WarningOptionalChecks {
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
		}
		return OptionalFail, nil
	}

	if !mandatoryCheck {
		return RequiredFail, nil
	} else if !optionalCheck {
		return OptionalFail, nil
	} else {
		return PASS, nil
	}

}
