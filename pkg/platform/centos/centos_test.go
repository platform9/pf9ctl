package centos

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/stretchr/testify/assert"
)

var errOSPackage = errors.New("Packages not found:  ntp curl")

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
		//Success case. Minimun CPU required is 2
		//Returning 4 CPUS. Therefore test case should pass
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
		//Failure case. CPUS should be less than 2 (No of CPU < 2)
		//Returning 1 CPUS. Therefore test case should pass
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
				err:    fmt.Errorf("Number of CPUs found: 1"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := &CentOS{exec: tc.exec}
			o, err := c.checkCPU()

			if diff := cmp.Diff(tc.want.result, o); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}

			assert.Equal(t, tc.err, err)
		})
	}
}

// RAM check test case
func TestRAM(t *testing.T) {
	type want struct {
		result bool
		err    error
	}

	cases := map[string]struct {
		args
		want
	}{
		//Success case. Minimum required RAM 12GB
		//Returning 12288 MB = 12 GB. Therefore test case should pass
		"CheckPass": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "12288", nil
					},
				},
			},
			want: want{
				result: true,
			},
		},
		//Failure case. RAM should be less than 12 GB. (RAM < 12 GB)
		//Returning 8 GB. Therefore test case should pass
		"CheckFail": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "8192", nil
					},
				},
			},
			want: want{
				result: false,
				err:    fmt.Errorf("Total memory found: 8 GB"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := &CentOS{exec: tc.exec}
			o, err := c.checkMem()

			if diff := cmp.Diff(tc.want.result, o); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}

			assert.Equal(t, tc.err, err)
		})
	}
}

// Disk check test case
func TestDisk(t *testing.T) {
	type want struct {
		result bool
		err    error
	}

	cases := map[string]struct {
		args
		want
	}{
		//Success case. Minimum required disk is 30 GB
		//Returning 31457280 KB = 30 GB. Therefore test case should pass.
		"CheckPass": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "31457280", nil
					},
				},
			},
			want: want{
				result: true,
			},
		},
		//Failure case. Disk should be less than 30 GB.
		//Returning 15728640 KB = 15 GB. Therefore test case should pass.
		"CheckFail": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "15728640", nil
					},
				},
			},
			want: want{
				result: false,
				err:    fmt.Errorf("Disk Space found: 15 GB"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := &CentOS{exec: tc.exec}
			o, err := c.checkDisk()

			if diff := cmp.Diff(tc.want.result, o); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}

			assert.Equal(t, tc.err, err)
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
		//Success case. User should have sudo permission.
		//If user id == 0 then user have sudo permission.
		//Returning 0. Therefore test case should pass.
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
		//Failure case. User should have id other than zero
		//Returning 100. Therefore test case should pass
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
			c := &CentOS{exec: tc.exec}
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
	}

	cases := map[string]struct {
		args
		want
	}{
		//Success case. Required ports should not be opened.
		//Returning ports which are not required. Therefore test case should pass.
		"CheckPass": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "22\n111\n25\n3036", nil
					},
				},
			},
			want: want{
				result: true,
			},
		},
		//Failure case. Required ports should be closed.
		//Returning closed ports. Therefore test case should pass
		"CheckFail": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "10255\n443\n10250", nil
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
			o, _ := c.checkPort()

			if diff := cmp.Diff(tc.want.result, o); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
		})
	}
}

//ExistingInstallation check test case
func TestExistingInstallation(t *testing.T) {
	type want struct {
		result bool
		err    error
	}

	cases := map[string]struct {
		args
		want
	}{
		//Success case. Packages should not be installed already.
		//If packages are not installed returning empty output. Therefore test case should pass.
		"CheckPass": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "", nil
					},
				},
			},
			want: want{
				result: true,
			},
		},
		//Failure case. If packages are installed already.
		//Returning list of packages. Therefore test case should pass.
		"CheckFail": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "pf9-comms.x86_64\npf9-hostagent.x86_64\npf9-kube.x86_64\npf9-muster.x86_64", nil
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
			o, err := c.checkExistingInstallation()

			if diff := cmp.Diff(tc.want.err, err); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.result, o); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestOSPackages(t *testing.T) {
	type want struct {
		result bool
		err    error
	}

	cases := map[string]struct {
		args
		want
	}{
		// Success case. OS Packages should be installed.
		"CheckPass": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRun: func(name string, args ...string) error {
						return nil
					},
				},
			},
			want: want{
				result: true,
			},
		},
		// Failure case. If packages are not installed.
		"CheckFail": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRun: func(name string, args ...string) error {
						return fmt.Errorf("Package not found")
					},
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "", fmt.Errorf("Error installing package")
					},
				},
			},
			want: want{
				result: false,
				err:    fmt.Errorf("%s %s", packageInstallError, strings.Join(packages, " ")),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := &CentOS{exec: tc.exec}
			o, err := c.checkOSPackages()

			if diff := cmp.Diff(tc.want.result, o); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}

			assert.Equal(t, tc.want.err, err)
		})
	}
}

//Test case for RemovePyCli check
func TestRemovePyCli(t *testing.T) {
	type want struct {
		result bool
		err    error
	}

	cases := map[string]struct {
		args
		want
	}{
		//Success case. Faking error code of rm -rf command. 0 error code indicates that rm -rf executed successfully
		//Returned nil error.
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
		//Failure case. Faking error code of rm -rf command. Other than 0 error code indicates some error occurred
		//Returned error value.
		"CheckFail": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "1", fmt.Errorf("Error")
					},
				},
			},
			want: want{
				result: false,
				err:    fmt.Errorf("Error"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := &CentOS{exec: tc.exec}
			result, err := c.removePyCli()

			assert.Equal(t, tc.want.err, err)
			assert.Equal(t, tc.want.result, result)
		})
	}
}

func TestCheckKubernetesCluster(t *testing.T) {
	type want struct {
		result bool
		err    error
	}

	cases := map[string]struct {
		args
		want
	}{
		//Success case. Node should not have any k8s cluster running
		//Returning 0 exit code. Therefore test case should pass.
		"CheckPass": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "0", nil
					},
				},
			},
			want: want{
				result: false,
				err:    k8sPresentError,
			},
		},
		//Failure case. If node running any k8s cluster.
		//Returning 1 exit status. Therefore test case should pass.
		"CheckFail": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "1", k8sPresentError
					},
				},
			},
			want: want{
				result: true,
				err:    nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := &CentOS{exec: tc.exec}
			o, err := c.checkKubernetesCluster()

			if diff := cmp.Diff(tc.want.result, o); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}

			assert.Equal(t, tc.err, err)
		})
	}
}

func TestCheckDocker(t *testing.T) {
	type want struct {
		err error
	}

	cases := map[string]struct {
		args
		want
	}{
		//Success case. Node should not have any container-runtime (Docker) running
		//Returning 0 exit code. Therefore test case should pass.
		"CheckPass": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "0", nil
					},
				},
			},
			want: want{
				err: k8sPresentError,
			},
		},
		//Failure case. If node running any container-runtime (Docker).
		//Returning 1 exit status. Therefore test case should pass.
		"CheckFail": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "1", k8sPresentError
					},
				},
			},
			want: want{
				err: k8sPresentError,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := &CentOS{exec: tc.exec}
			err := c.checkDocker()

			assert.Equal(t, tc.err, err)
		})
	}
}

func TestDisableSwap(t *testing.T) {
	type want struct {
		result bool
		err    error
	}

	cases := map[string]struct {
		args
		want
	}{
		//Success case. returns true on successful execution
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
				err:    nil,
			},
		},
		//Failure case. returns false if error occured while disabling swap.
		"CheckFail": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "1", errors.New("error occured while disabling swap")
					},
				},
			},
			want: want{
				result: false,
				err:    errors.New("error occured while disabling swap"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := &CentOS{exec: tc.exec}
			result, err := c.disableSwap()
			assert.Equal(t, tc.result, result)
			assert.Equal(t, tc.err, err)
		})
	}
}

func TestNoexecPermissionCheck(t *testing.T) {
	type want struct {
		result bool
		err    error
	}

	cases := map[string]struct {
		args
		want
	}{
		//Success case. successful execution of grep command returns error.
		"CheckPass": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "0", nil
					},
				},
			},
			want: want{
				result: false,
				err:    errors.New("/tmp is not having exec permission"),
			},
		},
		//Failure case. if output of grep command is empty then it returns nil error.
		"CheckFail": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "1", errors.New("ERROR")
					},
				},
			},
			want: want{
				result: true,
				err:    nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := &CentOS{exec: tc.exec}
			result, err := c.checkNoexecPermission()
			assert.Equal(t, tc.result, result)
			assert.Equal(t, tc.err, err)
		})
	}
}

func TestPIDofSystemdCheck(t *testing.T) {
	type want struct {
		result bool
		err    error
	}

	cases := map[string]struct {
		args
		want
	}{
		//Success case. if system is booted with systemd then its pid will be 1.
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
				err:    nil,
			},
		},
		//Failure case. if PID of systemd is not 1 then check will return error.
		"CheckFail": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "1", fmt.Errorf("ERROR")
					},
				},
			},
			want: want{
				result: false,
				err:    errors.New("System is not booted with systemd"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := &CentOS{exec: tc.exec}
			result, err := c.checkPIDofSystemd()
			assert.Equal(t, tc.result, result)
			assert.Equal(t, tc.err, err)
		})
	}
}

func TestCheckFirewalldService(t *testing.T) {
	type want struct {
		result bool
		err    error
	}

	cases := map[string]struct {
		args
		want
	}{
		//Success case. if firewalld service is not running then continue.
		"CheckPass": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "0", nil
					},
				},
			},
			want: want{
				result: false,
				err:    errors.New("firewalld service is running"),
			},
		},
		//Failure case. if firewalld service is running then bail out.
		"CheckFail": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "1", fmt.Errorf("ERROR")
					},
				},
			},
			want: want{
				result: true,
				err:    nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := &CentOS{exec: tc.exec}
			result, err := c.checkFirewalldIsRunning()
			assert.Equal(t, tc.result, result)
			assert.Equal(t, tc.err, err)
		})
	}
}
