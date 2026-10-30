package log

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/platform9/pf9ctl/pkg/util"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Returns the current log file location.
func GetLogLocation(logFile string) string {
	runLogLocation := fmt.Sprintf("%s-%s.%s", logFile[:strings.LastIndex(logFile, ".")], time.Now().Format("20060102"), logFile[strings.LastIndex(logFile, ".")+1:])
	return runLogLocation
}

// ConfigureGlobalLog global uber zap logger, there are two modes possible.
// the very first one is debug or not, used by CLI a debug mode is fantastic way
// to print more information and a logFile where the logs would be saved
func ConfigureGlobalLog(debug bool, logFile string) error {

	runLogLocation := GetLogLocation(logFile)
	util.LogFileNamePath = logFile
	// If the file doesn't exist, create it, or append to the file
	f, err := os.OpenFile(runLogLocation, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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
	core := maskingCore{zapcore.NewTee(
		zapcore.NewCore(zapcore.NewConsoleEncoder(consoleConfig()), consoleLogs, lvl),
		zapcore.NewCore(zapcore.NewJSONEncoder(fileConfig()), fileLogs, zap.DebugLevel),
	)}

	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	zap.ReplaceGlobals(logger)

	defer logger.Sync()
	return nil
}

// maskingCore runs every message and string/error field through util.Mask
// before the wrapped core writes it (PCD-9384). Commands, their output and
// errors are logged from many places; masking here covers all of them, on the
// console and in the log file.
type maskingCore struct {
	zapcore.Core
}

func (c maskingCore) With(fields []zapcore.Field) zapcore.Core {
	return maskingCore{c.Core.With(maskFields(fields))}
}

func (c maskingCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c maskingCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	ent.Message = util.Mask(ent.Message)
	return c.Core.Write(ent, maskFields(fields))
}

func maskFields(fields []zapcore.Field) []zapcore.Field {
	masked := make([]zapcore.Field, len(fields))
	for i, f := range fields {
		switch f.Type {
		case zapcore.StringType:
			f.String = util.Mask(f.String)
		case zapcore.ErrorType:
			if err, ok := f.Interface.(error); ok {
				f = zap.String(f.Key, util.Mask(err.Error()))
			}
		}
		masked[i] = f
	}
	return masked
}

func fileConfig() zapcore.EncoderConfig {
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.TimeEncoder(func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.UTC().Format("2006-01-02T15:04:05.9999Z"))
	})
	return config
}

func consoleConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		LevelKey:    "level",
		TimeKey:     "ts",
		MessageKey:  "msg",
		EncodeLevel: zapcore.CapitalLevelEncoder,
		EncodeTime: zapcore.TimeEncoder(func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.UTC().Format("2006-01-02T15:04:05.9999Z"))
		}),
	}
}
