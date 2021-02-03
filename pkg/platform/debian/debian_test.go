package debian

import (
	"testing"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/platform9/pf9ctl/pkg/cmdexec"
)

type args struct {
	exec cmdexec.Executor
}
//CPU check test case
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
			c := &Debian{exec: tc.exec}
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
//RAM check test case
func TestRAM(t *testing.T) {
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
						return "12000", nil
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
						return "8000", nil
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
			c := &Debian{exec: tc.exec}
			o, err := c.checkMem()

			if diff := cmp.Diff(tc.want.err, err); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.result, o); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
		})
	}
}
//Sudo check test case
func TestSudo(t *testing.T) {
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
						return "0", nil
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
						return "100", nil
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
			c := &Debian{exec: tc.exec}
			o, err := c.checkSudo()

			if diff := cmp.Diff(tc.want.err, err); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.result, o); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
		})
	}
}
//Port check test case
func TestPort(t *testing.T) {
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
						return "22\n23\n53\n44", nil
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
						return "10255\n443\n10255", nil
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
			c := &Debian{exec: tc.exec}
			o, err := c.checkPort()

			if diff := cmp.Diff(tc.want.err, err); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.result, o); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
		})
	}
}
//Disk check test case
func TestDisk(t *testing.T) {
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
						return "31000000", nil
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
						return "30000", nil
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
			c := &Debian{exec: tc.exec}
			o, err := c.checkDisk()

			if diff := cmp.Diff(tc.want.err, err); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.result, o); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
		})
	}
}
//Packages check test case
func TestPackages(t *testing.T) {
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
					MockRun: func(name string, args ...string) (error) {
						return nil
					},
				},
			},
			want: want{
				result: false,
			},
		},
		"CheckFail": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRun: func(name string, args ...string) (error) {
						return fmt.Errorf("Error")
					},
				},
			},
			want: want{
				result: true,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := &Debian{exec: tc.exec}
			o, err := c.checkPackages()
			
			if diff := cmp.Diff(tc.want.err, err); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.result, o); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
		})
	}
}