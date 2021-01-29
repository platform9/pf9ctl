package centos

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/platform9/pf9ctl/pkg/cmdexec"
)

type args struct {
	exec cmdexec.Executor
}

func TestCPU(t *testing.T) {
	type want struct {
		result bool
		err    error
	}

	cases := map[string]struct {
		args
		want
	}{
		"CheckPass": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "4", nil
					},
				},
			},
			want: want{
				result: true,
			},
		},
		"CheckFail": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "1", nil
					},
				},
			},
			want: want{
				result: false,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := &CentOS{exec: tc.exec}
			o, err := c.checkCPU()

			if diff := cmp.Diff(tc.want.err, err); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.result, o); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
		})
	}
}
