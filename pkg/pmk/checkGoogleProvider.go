// Copyright © 2020 The Platform9 Systems Inc.
package pmk

import (
	"fmt"

	context "golang.org/x/net/context"
	iamGoogle "google.golang.org/api/iam/v1"
	"google.golang.org/api/option"

	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/util"
)

func CheckGoogleProvider(path string) {

	//for the Google Cloud prerequisites the service app only has to have four roles given to it
	ctx := context.Background()

	//the user sends the path to json file
	iamService, err := iamGoogle.NewService(ctx, option.WithCredentialsFile(path))
	if err != nil {
		fmt.Println(err)
		return
	}

	//the four roles needed
	names := util.GoogleCloudPermissions

	//iterate through all roles needed so it can be upgraded easier later (if perhaps more roles have to be checked)
	for _, s := range names {

		resp, err := iamService.Projects.Roles.Get(s).Context(ctx).Do()
		if err != nil {
			fmt.Printf(color.Red("X ")+"%#v Failed\n", resp.Name)
		}

		fmt.Printf(color.Green("✓ ")+"%#v Success\n", resp.Name)
	}

}
