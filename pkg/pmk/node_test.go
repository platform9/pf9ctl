package pmk

import (
	"fmt"
	"testing"

	"github.com/platform9/pf9ctl/pkg/cmdexec"
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
			result, err := openOSReleaseFile(tc.args.exec)
			assert.Equal(t, tc.want.err, err)
			assert.Equal(t, tc.want.result, result)
		})
	}
}
