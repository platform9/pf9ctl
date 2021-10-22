// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
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
		fmt.Println("Pf9ctl version: " + util.Version + "\nChangelog:\n" + color.Green(util.Changelog))
	},
}

// versionCmd represents the version command
var upgrade = &cobra.Command{
	Use:   "upgrade",
	Short: "Checks for a new version of the CLI",
	Long:  "Checks and downloads the new version of the CLI. Use -c to skip the check and install the neweset version.",
	Run:   checkVersion,
}

func checkVersion(cmd *cobra.Command, args []string) {
	if skipCheck {
		err = upgradeVersion()
		if err != nil {
			zap.S().Fatalf(err.Error())
		}
		fmt.Println("Successfully updated, type pf9ctl version to check the changelog")
		return
	}
	// Code to compare the current version with the newest version
	newVersion, err := getNewestVersion()
	if err != nil {
		zap.S().Fatalf("Error getting the newest version")
	}
	if newVersion {
		fmt.Print("Do you want to upgrade?")
		answer, err := util.AskBool("")
		if err != nil {
			zap.S().Fatalf("Stopping upgrade")
		}
		if !answer {
			fmt.Println("Stopping upgrade")
			return
		}
		err = upgradeVersion()
		if err != nil {
			zap.S().Fatalf(err.Error())
		}
		fmt.Println("Successfully updated, type pf9ctl version to check the changelog")
	} else {
		fmt.Println("You already have the newest version")
	}
}

func getNewestVersion() (bool, error) {
	file, err := os.Open("/usr/bin/pf9ctl")
	if err != nil {
		zap.S().Fatalf("Error reading pr9ctl file", err.Error())
	}
	defer file.Close()
	hash := md5.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		zap.S().Fatalf("Error hashing pf9ctl file", err.Error())
	}
	hashString := hex.EncodeToString(hash.Sum(nil))
	eTag := getEtag()
	return !strings.Contains(eTag, hashString), nil

}

func getEtag() string {
	svc := s3.New(session.New(
		&aws.Config{
			Region:      aws.String(util.AWSBucketRegion),
			Credentials: credentials.AnonymousCredentials,
		}))
	input := &s3.GetObjectInput{
		Bucket: aws.String(util.AWSBucketName),
		Key:    aws.String(util.AWSBucketKey),
	}
	result, err := svc.GetObject(input)
	if err != nil {
		fmt.Errorf("Error while getting the neweset version " + err.Error())
	}
	return *result.ETag
}

func upgradeVersion() error {

	fmt.Println("\nDownloading the CLI")
	curlCmd, err := exec.Command("curl", "-sL", util.BucketPath).Output()
	if err != nil {
		return fmt.Errorf("Error downloading the setup " + err.Error())
	}
	bashCmd := exec.Command("bash", "-c", string(curlCmd))
	err = bashCmd.Start()
	fmt.Println("\nInstalling the CLI")
	bashCmd.Wait()
	if err != nil {
		return fmt.Errorf("Error installing the setup" + err.Error())
	}
	return nil

}

func checkVersionInit() {
	newVersion, err := getNewestVersion()
	if err != nil {
		zap.S().Fatalf("Error getting the newest version")
	}
	if newVersion {
		fmt.Println(color.Red("New version found. Please upgrade to the newest version"))
	}
}

func init() {

	cobra.OnInitialize(checkVersionInit)

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(upgrade)

	upgrade.Flags().BoolVarP(&skipCheck, "skipCheck", "c", false, "Will skip the version checks if true")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// versionCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// versionCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
