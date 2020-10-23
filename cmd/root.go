// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"fmt"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/platform9/pf9ctl/pkg/constants"
	"github.com/platform9/pf9ctl/pkg/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use: "pf9ctl",
	Long: `CLI tool for Platform9 management.
	Platform9 Managed Kubernetes cluster operations. Read more at
	http://pf9.io/cli_clhelp.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := initializeBaseDirs(); err != nil {
		fmt.Printf("Base directory initialization failed: %s\n", err.Error())
		os.Exit(1)
	}

	// Initializing zap log with console and file logging support
	if err := log.New(); err != nil {
		fmt.Printf("log initialization failed: %s", err.Error())
		os.Exit(1)
	}

	if err := rootCmd.Execute(); err != nil {
		log.Fatalf(err.Error())
	}
}

func initializeBaseDirs() (err error) {
	err = os.MkdirAll(constants.Pf9Dir, 0700)
	if err != nil {
		return 
	}
	err = os.MkdirAll(constants.Pf9DBDir, 0700)
	if err != nil {
		return
	}
	err = os.MkdirAll(constants.Pf9LogDir, 0700)
	return
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.pf9ctl.yaml)")
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// InitConfig reads in config file and ENV variables if set.
func initConfig() {

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		viper.AddConfigPath(home)
		viper.SetConfigName(".pf9ctl")
	}

	// Read in environment variables that match
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err == nil {
		log.Errorf("Error occured while reading the config file: %s", viper.ConfigFileUsed())
	}
}
