package util

import "testing"

func TestHostAgentStagesMatchBothArches(t *testing.T) {
	for _, line := range []string{
		"pf9-hostagent-2026.12.0-5311.2bce97c8.x86_64.deb",
		"pf9-hostagent-2026.12.0-5311.2bce97c8.aarch64.deb",
	} {
		if !HostAgentProgressBarStages[4].MatchString(line) {
			t.Errorf("hostagent stage did not match %q", line)
		}
	}
	for _, line := range []string{
		"pf9-comms-2026.12.0-1689.cb280fa.x86_64.deb",
		"pf9-comms-2026.12.0-1689.cb280fa.aarch64.deb",
	} {
		if !HostAgentProgressBarStages[5].MatchString(line) {
			t.Errorf("comms stage did not match %q", line)
		}
	}
}
