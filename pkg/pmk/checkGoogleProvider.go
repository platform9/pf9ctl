// Copyright © 2020 The Platform9 Systems Inc.
package pmk

import (
	"fmt"

	context "golang.org/x/net/context"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"

	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/util"
)

func CheckGoogleProvider(path, projectName, serviceAccountEmail string) bool {

	success := true
	//for the Google Cloud prerequisites the service app only has to have four roles given to it
	ctx := context.Background()

	iamService, err := iam.NewService(ctx, option.WithCredentialsFile(path))
	if err != nil {
		fmt.Println(err)
		return false
	}

	//the four roles needed
	names := util.GoogleCloudPermissions

	resource := "projects/" + projectName + "/serviceAccounts/" + serviceAccountEmail

	resp, err := iamService.Projects.ServiceAccounts.GetIamPolicy(resource).Context(ctx).Do()

	if err != nil || resp == nil {
		fmt.Printf(color.Red("X ")+"%#v Failed\n", err)
		return false
	}

	for _, name := range names {

		if !CheckIfRoleExists(resp.Bindings, name) {
			fmt.Println(color.Red("X ") + " Failed " + name)
			success = false
		} else {
			fmt.Println(color.Green("✓ ") + " Success " + name)
		}

	}

	return success

}

func CheckIfRoleExists(bindings []*iam.Binding, name string) bool {

	for _, binding := range bindings {

		if binding.Role == name {
			return true
		}

	}
	return false
}
