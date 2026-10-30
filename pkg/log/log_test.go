package log

import (
	"errors"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func newMaskedLogger(level zapcore.Level) (*zap.SugaredLogger, *observer.ObservedLogs) {
	core, logs := observer.New(level)
	return zap.New(maskingCore{core}).Sugar(), logs
}

func TestMaskingCore_MasksMessagesAndFields(t *testing.T) {
	log, logs := newMaskedLogger(zap.DebugLevel)

	log.Debugf("Ran command sudo %s", `"bash" "-c" "installer.sh --password='S3 cret' --proxy http://pf9:pr0xy@h:3128"`)
	log.Desugar().Info("stdout",
		zap.String("output", "password: --user-token=tok123"),
		zap.Error(errors.New("curl -H X-Auth-Token:abc123 failed")))
	log.Desugar().With(zap.String("cmd", "--password=S3cret")).Warn("retrying")

	entries := logs.AllUntimed()
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(entries))
	}
	var all strings.Builder
	for _, e := range entries {
		all.WriteString(e.Message + "\n")
		for k, v := range e.ContextMap() {
			all.WriteString(k + "=" + v.(string) + "\n")
		}
	}
	for _, secret := range []string{"S3 cret", "S3cret", "pr0xy", "tok123", "abc123"} {
		if strings.Contains(all.String(), secret) {
			t.Errorf("%q reached the log:\n%s", secret, all.String())
		}
	}
	if want := `--password=***** --proxy http://*****:*****@h:3128`; !strings.Contains(entries[0].Message, want) {
		t.Errorf("message = %q, want it to contain %q", entries[0].Message, want)
	}
	if got := entries[1].ContextMap()["error"]; got != "curl -H X-Auth-Token:***** failed" {
		t.Errorf("error field = %q, want the masked error text", got)
	}
}

// Negative case: lines with nothing secret pass through unchanged, and the
// wrapper doesn't let through levels the wrapped core has disabled.
func TestMaskingCore_LeavesOrdinaryLogsAlone(t *testing.T) {
	log, logs := newMaskedLogger(zap.InfoLevel)

	log.Debug("debug line the core does not enable")
	log.Infow("Hostagent download completed successfully", "host", "10.0.0.5")

	entries := logs.AllUntimed()
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want only the info one", len(entries))
	}
	if entries[0].Message != "Hostagent download completed successfully" || entries[0].ContextMap()["host"] != "10.0.0.5" {
		t.Errorf("ordinary entry changed: %+v", entries[0])
	}
}
