package cloud_provider

import (
	"encoding/json"
	"testing"

	assignment "github.com/Azure/azure-sdk-for-go/services/authorization/mgmt/2015-07-01/authorization"
	"github.com/platform9/pf9ctl/pkg/pmk"
	. "github.com/platform9/pf9ctl/pkg/test_utils"
)

var azureRolesInfoContributor string = `{

	"value": [
		{
		"properties": {

			"roleDefinitionId": "/subscriptions/6G3sg567b74/providers/Microsoft.Authorization/roleDefinitions/b24988ac-6180-42a0-ab88-20f7382dd24c"
		}
	}
	]
} `

var azureRolesInfoNonContributor string = `{

	"value": [
		{
		"properties": {

			"roleDefinitionId": "/subscriptions/6G3sg567b74/providers/Microsoft.Authorization/roleDefinitions/8e3af657-a8ff-443c-a75c-2fe8c4bcb635"
		}
	}
	]
} `

func TestAzureRole(t *testing.T) {

	rolesInfo := assignment.RoleAssignmentListResult{}

	//err := json.NewDecoder(ioutil.NopCloser(bytes.NewBufferString(azureRolesInfo))).Decode(&rolesInfo)

	err := json.Unmarshal([]byte(azureRolesInfoContributor), &rolesInfo)

	Ok(t, err)

	Equals(t, pmk.CheckRoleAssignment(rolesInfo, "6G3sg567b74"), true)
	Equals(t, pmk.CheckRoleAssignment(rolesInfo, "6G3sg567b77"), false)
	err = json.Unmarshal([]byte(azureRolesInfoNonContributor), &rolesInfo)
	Ok(t, err)
	Equals(t, pmk.CheckRoleAssignment(rolesInfo, "6G3sg567b74"), false)
	Equals(t, pmk.CheckRoleAssignment(rolesInfo, "6G3sg567b77"), false)

}
