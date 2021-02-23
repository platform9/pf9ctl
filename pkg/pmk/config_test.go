package pmk_test

import (
	"github.com/platform9/pf9ctl/cmd"
	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/stretchr/testify/assert"
	"testing"
)

//Load config test case
func TestLoadConfig(t *testing.T) {

	type args struct {
		err error
		loc string
	}
	testcases := map[string]struct {
		args
	}{
		//Success case. Check if config.json file present or not
		//Passing file location to open.
		"CheckPass": {
			args: args{
				loc: cmd.Pf9DBLoc,
				err: nil,
			},
		},
		//Failure case. Should return error for empty file name
		"CheckFail": {
			args: args{
				loc: "",
				err: pmk.ErrConfigurationDetailsNotProvided,
			},
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			_, err := pmk.LoadConfig(tc.args.loc)
			assert.Equal(t, tc.args.err, err)
		})
	}
}
