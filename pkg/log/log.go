package log

import (
	"fmt"
	"os"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)


// ConfigureGlobalLog global uber zap logger, there are two modes possible.
// the very first one is debug or not, used by CLI a debug mode is fantastic way
// to print more information and a logFile where the logs would be saved
func ConfigureGlobalLog(debug bool, logFile string) error {

	// If the file doesn't exist, create it, or append to the file
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("Couldn't open the log file: %s. \nError is: %s", logFile, err.Error())
	}

	var lvl zapcore.Level

	if debug {
		lvl = zap.DebugLevel
	} else {
		lvl = zap.InfoLevel
	}

	// Define log location
	consoleLogs := zapcore.Lock(os.Stderr)
	fileLogs := zapcore.Lock(f)

	// Create custom zap config
	core := zapcore.NewTee(
		zapcore.NewCore(zapcore.NewConsoleEncoder(setCustomConfig()), consoleLogs, lvl),
		zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), fileLogs, zap.DebugLevel),
	)

	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	zap.ReplaceGlobals(logger)

	defer logger.Sync()
	return nil
}

func setCustomConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		LevelKey:    "level",
		TimeKey:     "ts",
		MessageKey:  "msg",
		EncodeLevel: zapcore.CapitalLevelEncoder,
		EncodeTime:  zapcore.ISO8601TimeEncoder,
	}
}
