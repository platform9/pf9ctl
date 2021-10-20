package cloud_provider

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	iamAws "github.com/aws/aws-sdk-go/service/iam"
	"github.com/platform9/pf9ctl/pkg/pmk"
	. "github.com/platform9/pf9ctl/pkg/test_utils"
)

var amazonPermissionsInfo []*iamAws.EvaluationResult = []*iamAws.EvaluationResult{
	{
		EvalActionName:   aws.String("elasticloadbalancing:AddTags"),
		EvalDecision:     aws.String("allowed"),
		EvalResourceName: aws.String("*"),
	},
	{
		EvalActionName:   aws.String("elasticloadbalancing:ApplySecurityGroupsToLoadBalancer"),
		EvalDecision:     aws.String("allowed"),
		EvalResourceName: aws.String("*"),
	},
	{
		EvalActionName:   aws.String("elasticloadbalancing:DescribeTags"),
		EvalDecision:     aws.String("allowed"),
		EvalResourceName: aws.String("*"),
	},
	{
		EvalActionName:   aws.String("backup:CopyTarget"),
		EvalDecision:     aws.String("implicitDeny"),
		EvalResourceName: aws.String("*"),
	},
	{
		EvalActionName:   aws.String("test:Permission"),
		EvalDecision:     aws.String("denied"),
		EvalResourceName: aws.String("*"),
	},
}

func TestAmazonPermissions(t *testing.T) {

	//Will return false cause of the implicitDeny and denied flags
	Equals(t, pmk.CheckIfAllowed(amazonPermissionsInfo), false)

	//removing denied flag
	amazonPermissionsInfo = amazonPermissionsInfo[:len(amazonPermissionsInfo)-1]

	//will return false cause of the implicitDeny flag
	Equals(t, pmk.CheckIfAllowed(amazonPermissionsInfo), false)

	//removing implicitDeny flag
	amazonPermissionsInfo = amazonPermissionsInfo[:len(amazonPermissionsInfo)-1]

	//will return true since all EvalDecision flags are "allowed"
	Equals(t, pmk.CheckIfAllowed(amazonPermissionsInfo), true)
}
