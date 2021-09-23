// Copyright © 2020 The Platform9 Systems Inc.
package pmk

import (
	"fmt"
	"os"

	context "golang.org/x/net/context"

	assignment "github.com/Azure/azure-sdk-for-go/services/authorization/mgmt/2015-07-01/authorization"
	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/platform9/pf9ctl/pkg/color"
)

//////////////////////////////////////////

func CheckAzureProvider(tenantID, appID, subID, secretKey string) {

	//Gets environment values and saves them in temporary varaibles so they can be returned later
	oldAppID := os.Getenv("AZURE_CLIENT_ID")
	oldTenantID := os.Getenv("AZURE_TENANT_ID")
	oldSecretKey := os.Getenv("AZURE_CLIENT_SECRET")
	oldSubID := os.Getenv("AZURE_SUBSCRIPTION_ID")

	//Sets the new data as environment so that azure sdk can use it correctly
	os.Setenv("AZURE_CLIENT_ID", appID)
	os.Setenv("AZURE_CLIENT_SECRET", secretKey)
	os.Setenv("AZURE_TENANT_ID", tenantID)
	os.Setenv("AZURE_SUBSCRIPTION_ID", subID)

	ctx := context.TODO()

	client := assignment.NewRoleAssignmentsClient(subID)

	authorizer, err := auth.NewAuthorizerFromEnvironment()

	if err == nil {
		client.Authorizer = authorizer

	} else {
		fmt.Println(color.Red(" X ") + err.Error())
		return
	}

	//Gets the principalID of the application so that we can find the role of the service principal
	principalID, err := getPrincipalID(tenantID, appID)

	//Since no authorization is needed after getting principalID the environment values are set to old ones
	setEnvs(oldAppID, oldTenantID, oldSubID, oldSecretKey)

	if err != nil {
		fmt.Println(color.Red(" X ") + err.Error())
		return
	}

	//Gets the preparer for the list of principals with a filter
	request, err := client.ListPreparer(ctx, "principalId eq '"+principalID+"'")
	if err != nil {
		fmt.Println(color.Red(" X ") + err.Error())
		return
	}

	//Gets a sender for the list of principals using the preparer
	response, err := client.ListSender(request)

	if err != nil {
		fmt.Println(color.Red(" X ") + err.Error())
		return
	}

	//Gets a responder for the list of principals using the request
	result, err := client.ListResponder(response)

	if err != nil {
		fmt.Println(color.Red(" X ") + err.Error())
		return
	}

	//if the result.Value lengt is 0 that means the principal was never found
	if len(*result.Value) == 0 {
		fmt.Println(color.Red(" X ") + "Principal not found")
		return
	}

	if CheckRoleAssignment(result, subID) {
		fmt.Println(color.Green("\n✓ ") + "Has access")
	} else {
		fmt.Println(color.Red(" X ") + "Does not have access")
	}

}

func CheckRoleAssignment(result assignment.RoleAssignmentListResult, subID string) bool {

	fmt.Printf("%+v", result.Value)
	//Iterates through the list
	for _, s := range *result.Value {

		//The contributor role has a static ID so it checks the RoleDefiniitonID of the service principal by comparing it to
		//a string where only the subscriptionID is different
		if *s.Properties.RoleDefinitionID == "/subscriptions/"+subID+"/providers/Microsoft.Authorization/roleDefinitions/b24988ac-6180-42a0-ab88-20f7382dd24c" {
			return true
		}

	}

	return false

}

func setEnvs(appID, tenantID, subID, secretKey string) {

	//Sets environment values back to old ones
	os.Setenv("AZURE_CLIENT_ID", appID)
	os.Setenv("AZURE_CLIENT_SECRET", secretKey)
	os.Setenv("AZURE_TENANT_ID", tenantID)
	os.Setenv("AZURE_SUBSCRIPTION_ID", subID)

}

func getPrincipalID(tenantID string, appID string) (string, error) {

	ctx := context.Background()
	//creates a application client so it can get the principalID using applicationID
	client := graphrbac.NewApplicationsClient(tenantID)
	//this authorizer has to be used or else there will be "Token missmatch" or "Invalid audience" errors
	authorizer, err := auth.NewAuthorizerFromEnvironmentWithResource(azure.PublicCloud.ResourceIdentifiers.Graph)

	if err == nil {
		client.Authorizer = authorizer
	} else {
		return "ERROR", fmt.Errorf("App client error " + err.Error())
	}
	//Gets preparer, sender and responder
	request, err := client.GetServicePrincipalsIDByAppIDPreparer(ctx, appID)
	if err != nil {
		return "ERROR", fmt.Errorf("PrincipalIDPreparer Error " + err.Error())
	}

	response, err := client.GetServicePrincipalsIDByAppIDSender(request)

	if err != nil {
		return "ERROR", fmt.Errorf("PrincipalIDSender Error " + err.Error())
	}

	result, err := client.GetServicePrincipalsIDByAppIDResponder(response)

	if err != nil {
		return "ERROR", fmt.Errorf("PrincipalIDResponder Error " + err.Error())
	}

	//returns the value at the end
	return *result.Value, nil

}
