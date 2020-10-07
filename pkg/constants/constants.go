package constants

import (
	"os"
	"path/filepath"
	"time"
)

var (
	homeDir, _ = os.UserHomeDir()
	//Pf9Dir is the base pf9dir
	Pf9Dir = filepath.Join(homeDir, "pf9")
	//Pf9LogDir is the base path for creating log dir
	Pf9LogDir = filepath.Join(Pf9Dir, "log")
	// Pf9DBDir is the base dir for storing pf9 db context
	Pf9DBDir = filepath.Join(Pf9Dir, "db")
	// Pf9DBLoc represents location of the context file.
	Pf9DBLoc = filepath.Join(Pf9DBDir, "config.json")
	// Pf9Log represents location of the log.
	Pf9Log = filepath.Join(Pf9LogDir, "pf9ctl.log")
	// WaitPeriod is the sleep period for the cli
	// before it starts with the operations.
	WaitPeriod = time.Duration(60)
)
