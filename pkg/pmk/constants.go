package pmk

import (
	"os"
	"path/filepath"
)

var (
	homeDir, _ = os.UserHomeDir()
	pf9Dir     = filepath.Join(homeDir, "pf9")
	pf9LogDir  = filepath.Join(pf9Dir, "log")
	pf9DBDir   = filepath.Join(pf9Dir, "db")

	// Pf9DBLoc represents location of the context file.
	Pf9DBLoc = filepath.Join(pf9DBDir, "express.json")
	// Pf9Log represents location of the log.
	Pf9Log = filepath.Join(pf9LogDir, "pf9ctl.log")
)
