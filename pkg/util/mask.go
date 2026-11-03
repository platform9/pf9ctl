package util

import "regexp"

// MaskString replaces each credential Mask finds.
const MaskString = "*****"

// PCD-9384: credentials that reach log lines through command lines, installer
// output and proxy URLs. Keep in step with the host-side copies of these
// patterns: pcdctl's pkg/platform/debian/sudomask/pf9-log-mask (Ubuntu
// auth.log) and pcd-iso-builder's rocky/sudo-log-mask/pf9_sudo_mask.py (Rocky
// sudo.log).
var maskPatterns = []struct {
	re   *regexp.Regexp
	repl string
}{
	// --password=x, --password 'x y', --user-token="x", --proxy-pass x. A
	// quote may be backslash-escaped, as sudo writes it on Ubuntu.
	{regexp.MustCompile(`(--(?:password|user-token|proxy-pass)(?:=|\s+))(\\?'.*?\\?'|\\?".*?\\?"|\S+)`), "${1}" + MaskString},
	{regexp.MustCompile(`(?i)(X-Auth-Token:\s*)\S+`), "${1}" + MaskString},
	// scheme://user:pass@host -> scheme://*****:*****@host
	{regexp.MustCompile(`([A-Za-z][A-Za-z0-9+.-]*://)[^:/@\s]+:[^@\s]+@`), "${1}" + MaskString + ":" + MaskString + "@"},
	{regexp.MustCompile(`\b((?:OS_PASSWORD|OS_KEYSTONE_TOKEN)=)\S+`), "${1}" + MaskString},
}

// Mask returns s with every credential it recognises replaced by MaskString.
func Mask(s string) string {
	for _, p := range maskPatterns {
		s = p.re.ReplaceAllString(s, p.repl)
	}
	return s
}
