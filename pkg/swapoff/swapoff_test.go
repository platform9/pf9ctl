package swapoff

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/stretchr/testify/assert"
)

const content = `UUID=device_UUID none swap defaults 0 0
UUID=device_UUID none notswap defaults 0
UUID=device_UUID none notswap defaults
/swapfile none ext 0
UUID=device_UUID none nfs defaults 0 0
UUID=device_UUID none somethingelse defaults 0 0
#UUID=device_UUID none swap defaults 0 0
# UUID=device_UUID none swap defaults 0 0
  # UUID=device_UUID none swap defaults 0 0
  
`

// TestFsTabEdit tests the
func TestFsTabEdit(t *testing.T) {
	type want struct {
		err                 error
		expectedFileContent string
	}
	type args struct {
		exec cmdexec.Executor
	}
	cases := map[string]struct {
		args
		want
	}{
		//Success case, successfull substitution of linesin the file
		"CheckPass": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						o, err := exec.Command(name, args...).Output()
						return string(o), err
					},
				},
			},
			want: want{
				err:                 nil,
				expectedFileContent: "#" + content,
			},
		},
		//Failure case, file operation failure
		"CheckFail": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "", &os.PathError{}
					},
				},
			},
			want: want{
				err:                 &os.PathError{},
				expectedFileContent: content,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			tmpFile, err := ioutil.TempFile("/tmp", "pf9ctl_common_test")
			if err != nil {
				t.Errorf("Error creating a tempfile %s", err)
			}
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.Write([]byte(content))
			if err != nil {
				t.Errorf("Error writing to temp file %s", err)
			}
			err = tmpFile.Close()
			if err != nil {
				t.Errorf("Error writing to temp file %s", err)
			}
			defer os.Remove(tmpFile.Name() + ".bak")

			// Check the error, return in case of failure
			err = swapOffFstab(tc.args.exec, tmpFile.Name())
			assert.Equal(t, tc.want.err, err)

			readContentBytes, err := ioutil.ReadFile(tmpFile.Name())
			if err != nil {
				t.Errorf("Error reading the tmpFile after editing the fstab")
			}

			readContent := string(readContentBytes)

			// Check the file content
			if diff := cmp.Diff(tc.want.expectedFileContent, readContent); diff != "" {
				t.Errorf("r: -want, +got:\n%s", diff)
			}

		})
	}
}

//SwapOff test case
func TestSwapOff(t *testing.T) {
	type want struct {
		err error
	}
	type args struct {
		exec cmdexec.Executor
	}
	cases := map[string]struct {
		args
		want
	}{
		//Success case. The swapoff -a command returns nil error on successful execution
		"CheckPass": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "0", nil
					},
				},
			},
			want: want{
				err: nil,
			},
		},
		//Failure case. The swapoff -a command returns error on execution
		"CheckFail": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "1", fmt.Errorf("Error")
					},
				},
			},
			want: want{
				err: fmt.Errorf("Error"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := swapOff(tc.args.exec)
			assert.Equal(t, tc.want.err, err)
		})
	}
}
