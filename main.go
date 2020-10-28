// Copyright © 2020 The pf9ctl authors

package main

import (
	//"fmt"
	//"os"

	"github.com/platform9/pf9ctl/cmd"
)

func main() {
	// the program may run a machine different from the one where the installation is happening
	// so removing this check. If this check needs to happen it should happen where the installation is
	// happening
	// Check if program is run using root privileges
	//if os.Geteuid() != 0 {
	//	fmt.Println("This program requires root privileges. Please run the binary as a root user.")
	//	os.Exit(1)
	//}
	// Read the context variables.
	cmd.Execute()
}
