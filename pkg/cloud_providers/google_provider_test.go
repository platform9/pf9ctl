package cloud_provider

import (
	"encoding/json"
	"testing"

	"github.com/platform9/pf9ctl/pkg/pmk"
	. "github.com/platform9/pf9ctl/pkg/test_utils"
	"go.uber.org/zap"
	"google.golang.org/api/iam/v1"
)

var bindingsInfo string = `[
		{
			"role": "roles/iam.serviceAccountUser"
		},
		{
			"role": "roles/container.admin"
		},
		{
			"role": "roles/compute.viewer"
		},
		{
			"role": "roles/viewer"
		}
	]`

func TestRoles(t *testing.T) {

	iamBindings := []*iam.Binding{}

	err := json.Unmarshal([]byte(bindingsInfo), &iamBindings)
	Ok(t, err)

	if err != nil {
		zap.S().Errorf("Failed to decode endpoint information, Error: %s", err)
	}

	Equals(t, pmk.CheckIfRoleExists(iamBindings, "roles/iam.serviceAccountUser"), true)
	Equals(t, pmk.CheckIfRoleExists(iamBindings, "roles/container.admin"), true)
	Equals(t, pmk.CheckIfRoleExists(iamBindings, "roles/compute.viewer"), true)
	Equals(t, pmk.CheckIfRoleExists(iamBindings, "roles/viewerFake"), false)
}
