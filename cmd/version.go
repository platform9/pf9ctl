// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"fmt"
	"os"
	"os/exec"

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
		upgradeVersion()
		return
	}

	// Code to compare the current version with the newest version
	newVersion := getNewestVersion()

	if newVersion != util.Version {
		upgradeVersion()
	} else {
		fmt.Print("You already have the newest version\n")
	}

}

func getNewestVersion() string {
	return "pf9ctl version: v1.8"
}

func upgradeVersion() {

	curlCmd, err := exec.Command("curl", "-sL", "https://pmkft-assets.s3-us-west-1.amazonaws.com/pf9ctl_setup").Output()
	if err != nil {
		zap.S().Fatalf("Error downloading the setup ", err)
	}

	copyCmd := exec.Command("/bin/sh", "-c", "sudo mv /usr/bin/pf9ctl /usr/bin/pf9ctl_backup")
	err = copyCmd.Run()

	if err != nil {
		fmt.Println("Error creating backup", err)
	} else {
		fmt.Println("Backup successfully created")
	}

	bashCmd := exec.Command("bash", "-c", string(curlCmd))
	bashCmd.Stdout = os.Stdout
	err = bashCmd.Start()

	bashCmd.Wait()
	if err != nil {
		fmt.Println("Upgrade failed, reverting to backup")
		copyCmd := exec.Command("/bin/sh", "-c", "sudo mv /usr/bin/pf9ctl_backup /usr/bin/pf9ctl")
		err = copyCmd.Run()
		if err != nil {
			fmt.Println("Error restoring backup ", err)
		}
		zap.S().Fatalf("Error updating pf9ctl. ", err)
	}

	removeCmd := exec.Command("/bin/sh", "-c", "sudo rm /usr/bin/pf9ctl_backup")
	err = removeCmd.Run()
	if err != nil {
		fmt.Println("Error removing backup ", err)
	}

}

func init() {
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
