package cloud_provider

import (
	"encoding/json"
	"testing"

	"github.com/platform9/pf9ctl/pkg/pmk"
	. "github.com/platform9/pf9ctl/pkg/test_utils"
	"go.uber.org/zap"
	"google.golang.org/api/iam/v1"
)

var googleBindingsInfo string = `[
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

func TestGoogleRoles(t *testing.T) {

	iamBindings := []*iam.Binding{}

	err := json.Unmarshal([]byte(googleBindingsInfo), &iamBindings)
	Ok(t, err)

	if err != nil {
		zap.S().Errorf("Failed to decode endpoint information, Error: %s", err)
	}

	//will return true since all three roles are in the array in the response
	Equals(t, pmk.CheckIfRoleExists(iamBindings, "roles/iam.serviceAccountUser"), true)
	Equals(t, pmk.CheckIfRoleExists(iamBindings, "roles/container.admin"), true)
	Equals(t, pmk.CheckIfRoleExists(iamBindings, "roles/compute.viewer"), true)
	//will return false if the role is not in the array in the response
	Equals(t, pmk.CheckIfRoleExists(iamBindings, "roles/viewerFake"), false)
}
