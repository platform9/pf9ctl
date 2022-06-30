package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/supportBundle"
	"github.com/platform9/pf9ctl/pkg/test_utils"
)

//Errors returned by the functions
var ErrHostIP = errors.New("Host IP not found")
var ErrRemove = errors.New("Unable to remove bundle")
var ErrUpload = errors.New("Unable to upload supportBundle to S3")
var ErrPartialBundle = errors.New("Failed to generate complete supportBundle, generated partial bundle")

//HostIP test case
func TestHostIP(t *testing.T) {
	type want struct {
		host string
		err  error
	}

	type args struct {
		exec cmdexec.Executor
	}

	cases := map[string]struct {
		args
		want
	}{
		//Success case.The HostIP function returns nil error on successful execution
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
		//Failure case.The HostIP function returns ErrHostIP error on execution
		"CheckFail": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "1", ErrHostIP
					},
				},
			},
			want: want{
				err: ErrHostIP,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {

			_, err1 := supportBundle.HostIP(tc.exec)
			test_utils.Equals(t, tc.want.err, err1)
		})
	}

}

//GenSupportBundle test case(Only for Local Host)
func TestGenSupportBundle(t *testing.T) {

	//isRemote := false

	timestamp := time.Now()
	type want struct {
		targetfile string
		err        error
	}

	type args struct {
		exec      cmdexec.Executor
		timestamp time.Time
		isRemote  bool
	}

	cases := map[string]struct {
		args
		want
	}{
		//Success case.The GenSupportBundle function returns nil on successful execution
		"CheckCompleteBundle": {
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
		//Partial Bundle Generation Case.The GenSupportBundle function returns ErrPartialBundle error on execution
		"CheckPartialBundle": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "1", ErrPartialBundle
					},
				},
			},
			want: want{
				err: ErrPartialBundle,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {

			_, err1 := supportBundle.GenSupportBundle(tc.exec, timestamp, false)
			fmt.Printf("Printing error: %s %s", tc.want.err, err1)
			test_utils.Equals(t, tc.want.err, err1)

		})
	}

}

//GenTargetFilename test case
func TestGenTargetFilename(t *testing.T) {

	type want struct {
		targetfile string
	}
	//Mocking GenTargetFilename function to compare both the targetfiles generated
	hour := strconv.Itoa(supportBundle.Timestamp.Hour())
	minutes := strconv.Itoa(supportBundle.Timestamp.Minute())
	seconds := strconv.Itoa(supportBundle.Timestamp.Second())
	layout := supportBundle.Timestamp.Format("2006-01-02")
	tarname := "hostname" + "-" + layout + "-" + hour + "-" + minutes + "-" + seconds
	tarzipname := tarname + ".tar.gz"
	targetfile1 := "/tmp/" + tarzipname

	cases := map[string]struct {
		want
	}{
		//Success case.
		"CheckPass": {
			want: want{
				targetfile: targetfile1,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {

			targetfile := supportBundle.GenTargetFilename(supportBundle.Timestamp, "hostname")
			//Comparing both the targetfiles
			test_utils.Equals(t, tc.want.targetfile, targetfile)
		})
	}

}

//S3Upload test case
func TestS3Upload(t *testing.T) {
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
		//Success case.The S3Upload function returns nil error on successful execution
		"CheckPass": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRun: func(name string, args ...string) error {
						return nil
					},
				},
			},
			want: want{
				err: nil,
			},
		},
		//Failure case.The S3Upload function returns ErrUpload error on execution
		"CheckFail": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRun: func(name string, args ...string) error {
						return ErrUpload
					},
				},
			},
			want: want{
				err: ErrUpload,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {

			err1 := supportBundle.S3Upload(tc.exec)
			test_utils.Equals(t, tc.want.err, err1)
		})
	}

}

//RemoveBundle test case
func TestRemoveBundle(t *testing.T) {
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
		//Success case.The RemoveBundle function returns nil error on successful execution
		"CheckPass": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRun: func(name string, args ...string) error {
						return nil
					},
				},
			},
			want: want{
				err: nil,
			},
		},
		//Failure case.The RemoveBundle function returns ErrRemove error on execution
		"CheckFail": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRun: func(name string, args ...string) error {
						return ErrRemove
					},
				},
			},
			want: want{
				err: ErrRemove,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {

			err1 := supportBundle.RemoveBundle(tc.exec)
			test_utils.Equals(t, tc.want.err, err1)
		})
	}

}
