// Copyright © 2020 The pf9ctl authors

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// contextCmdCreate represents the context command
var contextCmdCreate = &cobra.Command{
	Use:   "context",
	Short: "Create a new context",
	Long:  `Create a new context that can be used to query Platform9 controller`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("context create called")
	},
}

// contextCmdCreate represents the context command
var contextCmdGet = &cobra.Command{
	Use:   "context",
	Short: "List stored context/s",
	Long:  `List stored contexts or details about a specific context`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("context get called")
	},
}

// contextCmdCreate represents the context command
var contextCmdUse = &cobra.Command{
	Use:   "context",
	Short: "Set a context to be used",
	Long:  `Use a new context`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("context use called")
	},
}

func init() {
	createCmd.AddCommand(contextCmdCreate)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// contextCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// contextCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	getCmd.AddCommand(contextCmdGet)
	useCmd.AddCommand(contextCmdUse)
}
