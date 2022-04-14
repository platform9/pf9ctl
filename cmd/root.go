// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	//homedir "github.com/mitchellh/go-homedir"
	"github.com/platform9/pf9ctl/pkg/log"
	"github.com/platform9/pf9ctl/pkg/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var cfgFile string
var verbosity bool
var detach bool
var logDirPath string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use: "pf9ctl",
	Long: `CLI tool for Platform9 management.
	Platform9 Managed Kubernetes cluster operations. Read more at
	http://pf9.io/cli_clhelp.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initializing zap log with console and file logging support
		if err := log.ConfigureGlobalLog(verbosity, util.Pf9Log); err != nil {
			return fmt.Errorf("log initialization failed: %s", err)
		}
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := initializeBaseDirs(); err != nil {
		// Fatalf does the same as Printf and os.Exit
		zap.S().Fatalf("Base directory initialization failed: %s\n", err.Error())
	}

	if err := rootCmd.Execute(); err != nil {
		zap.S().Fatalf(err.Error())
	}
}

func initializeBaseDirs() (err error) {
	err = os.MkdirAll(util.Pf9Dir, 0700)
	if err != nil {
		return
	}
	err = os.MkdirAll(util.Pf9DBDir, 0700)
	if err != nil {
		return
	}
	err = os.MkdirAll(util.Pf9LogDir, 0700)
	return
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().BoolVar(&verbosity, "verbose", false, "print verbose logs")
	rootCmd.PersistentFlags().BoolVar(&detach, "no-prompt", false, "disable all user prompts")
	rootCmd.PersistentFlags().StringVar(&logDirPath, "log-dir", "", "path to save logs")
	//rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.pf9ctl.yaml)")
	//rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// InitConfig reads in config file and ENV variables if set.
func initConfig() {
	/*if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		viper.AddConfigPath(home)
		viper.SetConfigName(".pf9ctl")
	}*/

	if rootCmd.Flags().Changed("log-dir") {
		logDirPath = filepath.Join(logDirPath)
		if _, err := os.Stat(logDirPath); os.IsNotExist(err) {
			zap.S().Debugf("%s dir is not present creating it", logDirPath)
			err = os.MkdirAll(logDirPath, 0700)
			if err != nil {
				zap.S().Fatalf("error creating %s dir, please make sure dir is present")
			}
		}
		util.Pf9Log = filepath.Join(logDirPath, "pf9ctl.log")
		util.Pf9LogLoc = logDirPath
	} else {
		util.Pf9LogLoc = util.DefaultPf9LogLoc
	}

	// Read in environment variables that match
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err == nil {
		zap.S().Errorf("Error occured while reading the config file: %s", viper.ConfigFileUsed())
	}
}
