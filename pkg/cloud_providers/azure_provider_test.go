package cloud_provider

import (
	"encoding/json"
	"testing"

	assignment "github.com/Azure/azure-sdk-for-go/services/authorization/mgmt/2015-07-01/authorization"
	"github.com/platform9/pf9ctl/pkg/pmk"
	. "github.com/platform9/pf9ctl/pkg/test_utils"
)

//One response where the service principal has contributor role
var azureRolesInfoContributor string = `{

	"value": [
		{
		"properties": {

			"roleDefinitionId": "/subscriptions/6G3sg567b74/providers/Microsoft.Authorization/roleDefinitions/b24988ac-6180-42a0-ab88-20f7382dd24c"
		}
	}
	]
} `

//one example where the service principal does not have the contributor role (has another one)
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
	//will return true since the subscription is correct and it has the contributor role
	Equals(t, pmk.CheckRoleAssignment(rolesInfo, "6G3sg567b74"), true)
	//will return false since the subcription does not match the one in the response
	Equals(t, pmk.CheckRoleAssignment(rolesInfo, "6G3sg567b77"), false)
	//now it will use the response with the wrong role
	err = json.Unmarshal([]byte(azureRolesInfoNonContributor), &rolesInfo)
	Ok(t, err)
	//will return false both times since the role is not contributor role, subscriptionid doesn't matter
	Equals(t, pmk.CheckRoleAssignment(rolesInfo, "6G3sg567b74"), false)
	Equals(t, pmk.CheckRoleAssignment(rolesInfo, "6G3sg567b77"), false)

}
