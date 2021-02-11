package pmk

import (
	"testing"
	"fmt"
	"github.com/stretchr/testify/assert"
)

//Load config test case
func TestLoadConfig(t *testing.T)  {
	
	type args struct{
		err error
		loc string
	}
	testcases := map[string]struct{
		args
	}{
		//Success case. Check if config.json file present or not
		//Passing file name to open.
		"CheckPass" : {
			args : args{
				loc : "config.json",
				err : nil,
			},
		},
		//Failure case. Should return error for empty file name
		"CheckFail" : {
			args : args{
				loc : "",
				err : fmt.Errorf("Config absent, run `sudo pf9ctl config set`"),
			},
		},
	}
	
	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			_, err := LoadConfig(tc.args.loc)
			assert.Equal(t, tc.args.err, err)
		})
	}
}