package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func NewCmdRoot() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sunpike",
		Short: "",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {

			// Setup logger
			logger, err := zap.NewDevelopment()
			if err != nil {
				return err
			}
			zap.ReplaceGlobals(logger)

			return nil
		},
	}

	cmd.AddCommand(NewCmdLogin())
	// TODO version cmd

	return cmd
}

func Execute() {
	if err := NewCmdRoot().Execute(); err != nil {
		os.Exit(1)
	}
}