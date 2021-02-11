package pmk

import (
	"testing"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
)

//Load config test case
func TestLoadConfig(t *testing.T)  {

	tmpFile, err := ioutil.TempFile("/tmp","config.json")
	defer os.Remove(tmpFile.Name()+".bak")
	if err != nil {
		t.Errorf("Error creating a tempfile %s", err)
	}
	
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
				loc : "/tmp",
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