// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

// prepNodeCmd represents the prepNode command
var prepNodeCmd = &cobra.Command{
	Use:   "prep-node",
	Short: "set up prerequisites & prep the node for k8s",
	Long: `Prepare a node to be ready to be added to a Kubernetes cluster. Read more
	at http://pf9.io/cli_clprep.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("prepNode called")
	},
}

func init() {
	rootCmd.AddCommand(prepNodeCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// prepNodeCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// prepNodeCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
func validate_platform() string {

	OS := runtime.GOOS
	if OS != "linux" {
		fmt.Println("Unsupported OS")
		os.Exit(0)
	} else {
		data, err := ioutil.ReadFile("/etc/os-release")
		if err != nil {
			log.Panicf("failed reading data from file: %s", err)
		}
		str_data := string(data)
		str_data_lower := strings.ToLower(str_data)

		if strings.Contains(str_data_lower, "centos") || strings.Contains(str_data_lower, "redhat") {
			out, err := exec.Command("bash", "-c", "cat /etc/*release | grep '(Core)' | grep 'CentOS Linux release' -m 1 | cut -f4 -d ' '").Output()
			if err != nil {
				log.Panicf("Couldn't read the OS configuration file os-release")
			}
			if strings.Contains(string(out), "7.6") || strings.Contains(string(out), "7.7") || strings.Contains(string(out), "7.8") {
				return "Supported"
			}
		} else {
			if strings.Contains(str_data_lower, "ubuntu") {
				out, err := exec.Command("bash", "-c", "cat /etc/*os-release | grep -i pretty_name | cut -d ' ' -f 2").Output()
				if err != nil {
					log.Panicf("Couldn't read the OS configuration file os-release")
				}
				if strings.Contains(string(out), "16") || strings.Contains(string(out), "18") {
					return "Supported"
				}

			}

		}
	}
	return ""
}
func common() {
	// Common tasks
}
