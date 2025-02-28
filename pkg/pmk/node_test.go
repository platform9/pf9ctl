package pmk

import (
	"fmt"
	"testing"

	"github.com/platform9/pf9ctl/pkg/client"
	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/keystone"
	"github.com/stretchr/testify/assert"
)

type args struct {
	exec cmdexec.Executor
}

func TestOpenOSReleaseFile(t *testing.T) {
	type want struct {
		result string
		err    error
	}

	cases := map[string]struct {
		args
		want
	}{
		//Success case. Mocking out Data of file.
		//Should return Data as lowercase string.
		"CheckPass": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "Data", nil
					},
				},
			},
			want: want{
				result: "data",
				err:    nil,
			},
		},
		//Mocking out empty output. Should return error saying that failed reading data from file.
		"CheckFail": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "", fmt.Errorf("Error")
					},
				},
			},
			want: want{
				result: "",
				err:    fmt.Errorf("failed reading data from file: Error"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := OpenOSReleaseFile(tc.args.exec)
			assert.Equal(t, tc.want.err, err)
			assert.Equal(t, tc.want.result, result)
		})
	}
}
func TestGetHostIDFromConf(t *testing.T) {
	type args struct {
		exec cmdexec.Executor
		auth keystone.KeystoneAuth
	}
	type want struct {
		hostID string
		err    error
	}

	cases := map[string]struct {
		args
		want
	}{
		"Success": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "host_id = xxxx-9688-46a2-aa4f-xxx", nil
					},
				},
				auth: keystone.KeystoneAuth{},
			},
			want: want{
				hostID: "xxxx-9688-46a2-aa4f-xxx",
				err:    nil,
			},
		},
		"ErrorGrep": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "", fmt.Errorf("stdout:stderr:grep: /etc/pf9/host_id.conf: No such file or directory")
					},
				},
				auth: keystone.KeystoneAuth{},
			},
			want: want{
				hostID: "",
				err:    fmt.Errorf("error: unable to grep host_id "),
			},
		},
		"EmptyOutput": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "", nil
					},
				},
				auth: keystone.KeystoneAuth{},
			},
			want: want{
				hostID: "",
				err:    fmt.Errorf("error: host_id not found in /etc/pf9/host_id.conf"),
			},
		},
		"InvalidFormat": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "invalid_format", nil
					},
				},
				auth: keystone.KeystoneAuth{},
			},
			want: want{
				hostID: "",
				err:    fmt.Errorf("error: host_id key=value pair not found in config"),
			},
		},
		"NoValue": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "host_id = ", nil
					},
				},
				auth: keystone.KeystoneAuth{},
			},
			want: want{
				hostID: "",
				err:    fmt.Errorf("error: no host_id value found in config"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			hostID, err := getHostIDFromConf(client.Client{Executor: tc.args.exec, Segment: client.NewSegment("mgain.pf9.test", false)}, tc.args.auth)
			assert.Equal(t, tc.want.err, err)
			assert.Equal(t, tc.want.hostID, hostID)
		})
	}
}
