// Copyright © 2020 The pf9ctl authors

package main

import (
	"fmt"
	"os"

	"github.com/platform9/pf9ctl/cmd"
)

func main() {
	// Check if program is run using root privileges
	if os.Geteuid() != 0 {
		fmt.Println("This program requires root privileges. Please run the binary as a root user.")
		os.Exit(1)
	}
	// Read the context variables.
	cmd.Execute()
}
