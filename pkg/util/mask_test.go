package util

import "testing"

func TestMask_MasksEachKindOfCredential(t *testing.T) {
	cases := []struct{ in, want string }{
		{`installer.sh --password=S3cret --user-token=tok123`,
			`installer.sh --password=***** --user-token=*****`},
		{`installer.sh --password 'pw with spaces' --proxy-pass "x y"`,
			`installer.sh --password ***** --proxy-pass *****`},
		// sudo on Ubuntu escapes quotes inside a logged command.
		{`COMMAND=/usr/bin/bash -c 'installer.sh --password=\'S3 cret\' --user-token=\'tok123\''`,
			`COMMAND=/usr/bin/bash -c 'installer.sh --password=***** --user-token=*****'`},
		{`curl -H X-Auth-Token:abc123 https://du`, `curl -H X-Auth-Token:***** https://du`},
		{`installer.sh --proxy https://u:p%40ss@h:3128`, `installer.sh --proxy https://*****:*****@h:3128`},
		{`env OS_PASSWORD=S3cret OS_KEYSTONE_TOKEN=tok123 installer.sh`,
			`env OS_PASSWORD=***** OS_KEYSTONE_TOKEN=***** installer.sh`},
		// A command line as cmdexec logs it: every argument double-quoted, the
		// proxy passed as an https_proxy= argument to sudo.
		{` "https_proxy=http://pf9:pr0xy@10.0.0.1:3128" "bash" "-c" "/root/pf9/installer.sh --proxy http://pf9:pr0xy@10.0.0.1:3128 --no-project --password='S3 cret' --insecure"`,
			` "https_proxy=http://*****:*****@10.0.0.1:3128" "bash" "-c" "/root/pf9/installer.sh --proxy http://*****:*****@10.0.0.1:3128 --no-project --password=***** --insecure"`},
	}
	for _, c := range cases {
		if got := Mask(c.in); got != c.want {
			t.Errorf("Mask(%q)\n got %q\nwant %q", c.in, got, c.want)
		}
	}
}

func TestMask_LeavesNonSecretTextUntouched(t *testing.T) {
	for _, s := range []string{
		`installer.sh --username=pf9 --password-file=/root/pw --controller=du.example.com`,
		`installer.sh --proxy http://10.0.0.1:3128 --no-ntp`,
		`Hostagent download completed successfully`,
		``,
	} {
		if got := Mask(s); got != s {
			t.Errorf("Mask(%q) = %q, want it unchanged", s, got)
		}
	}
}
