package cmd

import (
	"testing"

	"github.com/platform9/pf9ctl/pkg/cmdexec"
	"github.com/platform9/pf9ctl/pkg/test_utils"
)

func TestHostID(t *testing.T) {
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
		//Success case.The HostId function returns nil error on successful execution
		"CheckPass": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "hostid", nil
					},
				},
			},
			want: want{
				err: nil,
			},
		},
		//Failure case.The HostIP function returns error if failss
		"CheckFail": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "", Errhostid
					},
				},
			},
			want: want{
				err: Errhostid,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {

			_, err := hostId(tc.exec, "Fqdn", "token", masterIPs)
			test_utils.Equals(t, tc.want.err, err)
		})
	}

}

func TestClusterStatus(t *testing.T) {
	type want struct {
		status string
	}

	type args struct {
		exec cmdexec.Executor
	}

	cases := map[string]struct {
		args
		want
	}{
		//Success case.The cluster_status function returns ok if cluster is ready
		"CheckPass": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "ok", nil
					},
				},
			},
			want: want{
				status: "ok",
			},
		},
		//Failure case.The cluster_status function returns pending if cluster is not ready
		"CheckFail": {
			args: args{
				exec: &cmdexec.MockExecutor{
					MockRunWithStdout: func(name string, args ...string) (string, error) {
						return "pending", nil
					},
				},
			},
			want: want{
				status: "pending",
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {

			status := cluster_Status(tc.exec, "Fqdn", "token", "projectid", "clusterid")
			test_utils.Equals(t, tc.want.status, status)
		})
	}

}
