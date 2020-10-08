package log

import (
	"fmt"
	"os"

	"github.com/platform9/pf9ctl/pkg/constants"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var s *zap.SugaredLogger

func New() error {
	// If the file doesn't exist, create it, or append to the file
	f, err := os.OpenFile(constants.Pf9Log, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("Couldn't open the log file: %s. \nError is: %s", constants.Pf9Log, err.Error())
	}

	// Define log location
	consoleLogs := zapcore.Lock(os.Stdout)
	fileLogs := zapcore.Lock(f)

	// Create custom zap config
	core := zapcore.NewTee(
		zapcore.NewCore(zapcore.NewConsoleEncoder(setCustomConfig()), consoleLogs, zap.InfoLevel),
		zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), fileLogs, zap.DebugLevel),
	)

	s = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1)).Sugar()

	defer s.Sync()
	return nil
}

func setCustomConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		LevelKey:    "level",
		TimeKey:     "ts",
		MessageKey:  "msg",
		EncodeLevel: zapcore.CapitalLevelEncoder,
		EncodeTime:  zapcore.EpochTimeEncoder,
	}
}
