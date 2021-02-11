package pmk
import (
	"testing"
	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"io/ioutil"
	"os"
	"go.uber.org/zap"
	"github.com/stretchr/testify/assert"
	"fmt"
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

func TestFsTabEdit(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)

	expectedContent := "#" + content
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

	// also remove the .bak file we create
	defer os.Remove(tmpFile.Name()+".bak")

	executor := cmdexec.LocalExecutor{}
	err = swapOffFstab(executor, tmpFile.Name())
	if err != nil {
		t.Errorf("Error editing fstab %s", err)
	}
	
	// now read the file and compare the content

	readContentBytes, err := ioutil.ReadFile(tmpFile.Name())
	if err != nil {
		t.Errorf("Error reading the tmpFile after editing the fstab")
	}

	readContent := string(readContentBytes)
	
	if readContent != expectedContent {
		t.Log("Expected:", expectedContent)
		t.Log("ReadContent:", readContent)
		t.Errorf("Test failed,content mistmatch")
	}

}

//SwapOff test case
func TestSwapOff(t *testing.T) {
	type want struct {
		err    error
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
				err : nil,
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
				err : fmt.Errorf("Error"),
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
