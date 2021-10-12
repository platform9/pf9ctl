// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/util"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var skipCheck bool

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Current version of CLI being used",
	Long:  "Gives the current pf9ctl version",
	Run: func(cmd *cobra.Command, args []string) {
		zap.S().Debug("Version called")
		//Prints the current version of pf9ctl being used.
		fmt.Println(util.Version)
	},
}

// versionCmd represents the version command
var upgrade = &cobra.Command{
	Use:   "upgrade",
	Short: "Checks for a new version of the CLI",
	Long:  "Checks and downloads the new version of the CLI. Use -skipCheck to just download the newest version.",
	Run:   checkVersion,
}

func checkVersion(cmd *cobra.Command, args []string) {

	if skipCheck {
		_, changelog, err := getNewestVersion()
		if err != nil {
			zap.S().Fatalf("Error getting the newest verison")
		}
		err = upgradeVersion()
		if err != nil {
			zap.S().Fatalf(err.Error())
		}
		fmt.Println("Changelog: \n" + color.Green(changelog))
		return
	}

	// Code to compare the current version with the newest version
	newVersion, changelog, err := getNewestVersion()

	if err != nil {
		zap.S().Fatalf("Error getting the newest verison")
	}

	if newVersion != util.Version {

		fmt.Print("New version found, your version is ", color.Red(util.Version)+" but newest version is "+color.Green(newVersion)+"\nDo you want to upgrade?")
		answer, err := util.AskBool("")

		if err != nil {
			zap.S().Fatalf("Stopping upgrade ")
		}

		if !answer {
			fmt.Println("Stopping upgrade ")
		}

		err = upgradeVersion()
		if err != nil {
			zap.S().Fatalf(err.Error())
		}

		fmt.Println("Changelog: \n" + color.Green(changelog))
	} else {
		fmt.Println("You already have the newest version")
	}

}

func getNewestVersion() (string, string, error) {

	curlCmd, err := exec.Command("curl", "-sL", util.VersionPath).Output()

	if err != nil {
		return "", "", err
	}

	version := strings.Split(string(curlCmd), "\n")[0]

	return version, string(curlCmd), nil
}

func upgradeVersion() error {

	fmt.Println("\nDownloading the CLI")

	curlCmd, err := exec.Command("curl", "-sL", "https://pmkft-assets.s3-us-west-1.amazonaws.com/pf9ctl_setup").Output()
	if err != nil {
		return fmt.Errorf("Error downloading the setup " + err.Error())
	}

	err = os.Rename("/usr/bin/pf9ctl", "/usr/bin/pf9ctl_backup")
	if err != nil {
		fmt.Println("Error creating backup\n", err)
	} else {
		fmt.Println("\nBackup successfully created")
	}

	bashCmd := exec.Command("bash", "-c", string(curlCmd))
	err = bashCmd.Start()

	fmt.Println("\nInstalling the CLI")

	bashCmd.Wait()
	if err != nil {
		fmt.Println("\nUpgrade failed, reverting to backup")
		err = os.Rename("/usr/bin/pf9ctl_backup", "/usr/bin/pf9ctl")
		if err != nil {
			fmt.Println("\nError restoring backup ", err)
		} else {
			fmt.Println("\nBackup successfully restored")
		}
		return fmt.Errorf("Error updating pf9ctl. " + err.Error())
	}

	err = os.Remove("/usr/bin/pf9ctl_backup")
	if err != nil {
		fmt.Println("Error removing backup ", err)
	}
	return nil

}

func checkVersionInit() {
	newVersion, _, err := getNewestVersion()

	if err != nil {
		zap.S().Fatalf("Error getting the newest verison")
	}

	if newVersion != util.Version {
		fmt.Print("\nNew version found, your version is ", color.Red(util.Version)+" but newest version is "+color.Green(newVersion)+"\nPlease run 'sudo pf9ctl upgrade' to have the newest version\n")
	}
}

func init() {

	cobra.OnInitialize(checkVersionInit)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(upgrade)

	upgrade.Flags().BoolVarP(&skipCheck, "skipCheck", "c", false, "Will skip the verison checks if true")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// versionCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// versionCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
