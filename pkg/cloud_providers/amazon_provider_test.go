package cloud_provider

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	iamAws "github.com/aws/aws-sdk-go/service/iam"
	"github.com/platform9/pf9ctl/pkg/pmk"
	. "github.com/platform9/pf9ctl/pkg/test_utils"
)

var permissionInfo []*iamAws.EvaluationResult = []*iamAws.EvaluationResult{
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

func TestPermissions(t *testing.T) {

	Equals(t, pmk.CheckIfAllowed(permissionInfo), false)

	permissionInfo = permissionInfo[:len(permissionInfo)-1]

	Equals(t, pmk.CheckIfAllowed(permissionInfo), false)

	permissionInfo = permissionInfo[:len(permissionInfo)-1]

	Equals(t, pmk.CheckIfAllowed(permissionInfo), true)
}
