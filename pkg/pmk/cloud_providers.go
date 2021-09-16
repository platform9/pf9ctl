// Copyright © 2020 The Platform9 Systems Inc.
package pmk

import (
	"fmt"
	"os"

	context "golang.org/x/net/context"
	iamGoogle "google.golang.org/api/iam/v1"
	"google.golang.org/api/option"

	assignment "github.com/Azure/azure-sdk-for-go/services/authorization/mgmt/2015-07-01/authorization"
	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	iamAws "github.com/aws/aws-sdk-go/service/iam"
)

func CheckGoogleProvider(path string) error {

	//for the Google Cloud prerequisites the service app only has to have four roles given to it
	ctx := context.Background()

	//the user sends the path to json file
	iamService, err := iamGoogle.NewService(ctx, option.WithCredentialsFile(path))
	if err != nil {
		return fmt.Errorf(err.Error())
	}

	//the four roles needed
	names := []string{
		"roles/iam.serviceAccountUser",
		"roles/container.admin",
		"roles/compute.viewer",
		"roles/viewer",
	}

	//iterate through all roles needed so it can be upgraded easier later (if perhaps more roles have to be checked)
	for _, s := range names {

		resp, err := iamService.Projects.Roles.Get(s).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("Unable check google provider %w", err)
		}

		fmt.Printf("%#v Success\n", resp.Name)
	}

	return nil

}

//////////////////////////////////////////////

func CheckAmazonPovider(username, awsID, awsSecret string) error {

	//creates a new session using static credentials that the user passes and "" stands for token that is of no use in this situation
	svc := iamAws.New(session.New(
		&aws.Config{
			Region:      aws.String("us-east-1"),
			Credentials: credentials.NewStaticCredentials(awsID, awsSecret, ""),
		}))

	//creates user input using username that is passed
	inputUser := &iamAws.GetUserInput{
		UserName: aws.String(username),
	}

	//gets the user with the passed username
	resultUser, errUser := svc.GetUser(inputUser)
	if errUser != nil {
		return fmt.Errorf("Amazon user error " + errUser.Error())
	}
	arn := resultUser.User.Arn

	//checkpermission function takes the arn, svc and an array of permissions needed and checks if the user has all of them
	//this can be easily upgraded by just copy pasting one checkPermission block of code and changing the permission array
	if checkPermissions(arn, svc, []string{
		"elasticloadbalancing:AddTags",
		"elasticloadbalancing:ApplySecurityGroupsToLoadBalancer",
		"elasticloadbalancing:AttachLoadBalancerToSubnets",
		"elasticloadbalancing:ConfigureHealthCheck",
		"elasticloadbalancing:CreateLoadBalancer",
		"elasticloadbalancing:CreateLoadBalancerListeners",
		"elasticloadbalancing:DeleteLoadBalancer",
		"elasticloadbalancing:DescribeLoadBalancerAttributes",
		"elasticloadbalancing:DescribeLoadBalancers",
		"elasticloadbalancing:DescribeTags",
		"elasticloadbalancing:ModifyLoadBalancerAttributes",
		"elasticloadbalancing:RemoveTags",
	}) == true {
		fmt.Println("ELB Access")
	} else {
		return fmt.Errorf("ELB Access Error")
	}

	if checkPermissions(arn, svc, []string{
		"route53:ChangeResourceRecordSets",
		"route53:GetChange",
		"route53:GetHostedZone",
		"route53:ListHostedZones",
		"route53:ListResourceRecordSets",
	}) == true {
		fmt.Println("Route53 Access")
	} else {
		return fmt.Errorf("Route53 Access Error")
	}

	//gets all of the users availability zones
	zoneSess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	zoneSvc := ec2.New(zoneSess)

	resultAvalZones, err := zoneSvc.DescribeAvailabilityZones(nil)
	if err != nil {
		return fmt.Errorf("Availability Zones error", err)
	}

	//checks if the user less than 2 availability zones
	if len(resultAvalZones.AvailabilityZones) < 2 {
		return fmt.Errorf("Availability Zones error")

	}
	fmt.Println("Availability Zones success")

	if checkPermissions(arn, svc, []string{
		"ec2:AllocateAddress",
		"ec2:AssociateRouteTable",
		"ec2:AttachInternetGateway",
		"ec2:AuthorizeSecurityGroupEgress",
		"ec2:AuthorizeSecurityGroupIngress",
		"ec2:CreateInternetGateway",
		"ec2:CreateNatGateway",
		"ec2:CreateRoute",
		"ec2:CreateRouteTable",
		"ec2:CreateSecurityGroup",
		"ec2:CreateSubnet",
		"ec2:CreateTags",
		"ec2:DeleteInternetGateway",
		"ec2:DeleteNatGateway",
		"ec2:DeleteRoute",
		"ec2:DeleteRouteTable",
		"ec2:DeleteSecurityGroup",
		"ec2:DeleteSubnet",
		"ec2:DeleteTags",
		"ec2:DescribeAccountAttributes",
		"ec2:DescribeAddresses",
		"ec2:DescribeAvailabilityZones",
		"ec2:DescribeImages",
		"ec2:DescribeInstances",
		"ec2:DescribeInternetGateways",
		"ec2:DescribeKeyPairs",
		"ec2:DescribeNatGateways",
		"ec2:DescribeNetworkAcls",
		"ec2:DescribeNetworkInterfaces",
		"ec2:DescribeRegions",
		"ec2:DescribeRouteTables",
		"ec2:DescribeSecurityGroups",
		"ec2:DescribeSubnets",
		"ec2:DetachInternetGateway",
		"ec2:DisassociateRouteTable",
		"ec2:ImportKeyPair",
		"ec2:ModifySubnetAttribute",
		"ec2:ReleaseAddress",
		"ec2:ReplaceRouteTableAssociation",
		"ec2:RevokeSecurityGroupEgress",
		"ec2:RevokeSecurityGroupIngress",
		"ec2:RunInstances",
		"ec2:TerminateInstances",
	}) == true {
		fmt.Println("EC2 Access")
	} else {
		return fmt.Errorf("EC2 Access Error")

	}

	if checkPermissions(arn, svc, []string{
		"ec2:CreateVpc",
		"ec2:DeleteVpc",
		"ec2:DescribeVpcAttribute",
		"ec2:DescribeVpcClassicLink",
		"ec2:DescribeVpcClassicLinkDnsSupport",
		"ec2:DescribeVpcs",
		"ec2:ModifyVpcAttribute",
	}) == true {
		fmt.Println("VPC Access")
	} else {
		return fmt.Errorf("VPC Access Error")
	}

	if checkPermissions(arn, svc, []string{
		"iam:AddRoleToInstanceProfile",
		"iam:CreateInstanceProfile",
		"iam:CreateRole",
		"iam:CreateServiceLinkedRole",
		"iam:DeleteInstanceProfile",
		"iam:DeleteRole",
		"iam:DeleteRolePolicy",
		"iam:GetInstanceProfile",
		"iam:GetRole",
		"iam:GetRolePolicy",
		"iam:GetUser",
		"iam:ListAttachedRolePolicies",
		"iam:ListInstanceProfilesForRole",
		"iam:ListRolePolicies",
		"iam:PassRole",
		"iam:PutRolePolicy",
		"iam:RemoveRoleFromInstanceProfile",
	}) == true {
		fmt.Println("IAM Access")
	} else {
		return fmt.Errorf("IAM Access Error")
	}

	if checkPermissions(arn, svc, []string{
		"autoscaling:AttachLoadBalancers",
		"autoscaling:CreateAutoScalingGroup",
		"autoscaling:CreateLaunchConfiguration",
		"autoscaling:CreateOrUpdateTags",
		"autoscaling:DeleteAutoScalingGroup",
		"autoscaling:DeleteLaunchConfiguration",
		"autoscaling:DeleteTags",
		"autoscaling:DescribeAutoScalingGroups",
		"autoscaling:DescribeLaunchConfigurations",
		"autoscaling:DescribeLoadBalancers",
		"autoscaling:DescribeScalingActivities",
		"autoscaling:DetachLoadBalancers",
		"autoscaling:EnableMetricsCollection",
		"autoscaling:UpdateAutoScalingGroup",
		"autoscaling:SuspendProcesses",
		"autoscaling:ResumeProcesses",
		"elasticloadbalancing:DescribeInstanceHealth",
	}) == true {
		fmt.Println("Autoscaling Access")
	} else {
		return fmt.Errorf("Autoscaling Access Error")

	}

	if checkPermissions(arn, svc, []string{
		"eks:ListAddons",
		"eks:ListClusters",
		"eks:ListFargateProfiles",
		"eks:ListIdentityProviderConfigs",
		"eks:ListNodegroups",
		"eks:ListUpdates",
		"eks:AccessKubernetesApi",
		"eks:DescribeAddon",
		"eks:DescribeAddonVersions",
		"eks:DescribeCluster",
		"eks:DescribeFargateProfile",
		"eks:DescribeIdentityProviderConfig",
		"eks:DescribeNodegroup",
		"eks:DescribeUpdate",
		"eks:ListTagsForResource",
		"eks:TagResource",
		"eks:UntagResource",
		"eks:AssociateEncryptionConfig",
		"eks:AssociateIdentityProviderConfig",
		"eks:CreateAddon",
		"eks:CreateCluster",
		"eks:CreateFargateProfile",
		"eks:CreateNodegroup",
		"eks:DeleteAddon",
		"eks:DeleteCluster",
		"eks:DeleteFargateProfile",
		"eks:DeleteNodegroup",
		"eks:DisassociateIdentityProviderConfig",
		"eks:UpdateAddon",
		"eks:UpdateClusterConfig",
		"eks:UpdateClusterVersion",
		"eks:UpdateNodegroupConfig",
		"eks:UpdateNodegroupVersion",
	}) == true {
		fmt.Println("EKS Access")
	} else {
		return fmt.Errorf("EKS Access Error")
	}

	return nil

}

func checkPermissions(arn *string, svc *iamAws.IAM, actions []string) bool {

	//turns the array of strings into an array of pointers
	//this is done so it is easier to call checkpermissions since permissions can be pasted as strings
	//otherwise there would need to be more code to call this function
	actionNames := getActionNames(actions)

	//simulate gets all permissions from the actionNames array
	input := &iamAws.SimulatePrincipalPolicyInput{
		PolicySourceArn: arn,
		ActionNames:     actionNames,
	}

	result, err := svc.SimulatePrincipalPolicy(input)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}
	//and then checks if the user is allowed to use them by calling checkArray
	if checkArray(result.EvaluationResults) == false {
		fmt.Println("Access Error")
		return false
	}

	return true

}

func checkArray(results []*iamAws.EvaluationResult) bool {

	//takes an array of user permissions and checks if the EvalDecision flag is not equal to allowed in which case the user doenst have
	//the permisison
	for i := range results {
		if *results[i].EvalDecision != "allowed" {
			return false
		}
	}
	return true

}

func getActionNames(actions []string) []*string {

	//turns array of strings into array of pointers
	var actionNames []*string
	for i := range actions {
		actionNames = append(actionNames, &actions[i])
	}
	return actionNames

}

//////////////////////////////////////////

func CheckAzureProvider(tenantID, appID, subID, secretKey string) error {

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
		return fmt.Errorf("No Access " + err.Error())
	}

	//Gets the principalID of the application so that we can find the role of the service principal
	principalID, err := getPrincipalID(tenantID, appID)

	//Since no authorization is needed after getting principalID the environment values are set to old ones
	returnEnv(oldAppID, oldTenantID, oldSubID, oldSecretKey)

	if err != nil {
		return fmt.Errorf("Error getting PrincipalID " + err.Error())
	}

	//Gets the preparer for the list of principals with a filter
	request, err := client.ListPreparer(ctx, "principalId eq '"+principalID+"'")
	if err != nil {
		return fmt.Errorf("No Access " + err.Error())
	}

	//Gets a sender for the list of principals using the preparer
	response, err := client.ListSender(request)

	if err != nil {
		return fmt.Errorf("No Access " + err.Error())
	}

	//Gets a responder for the list of principals using the request
	result, err := client.ListResponder(response)

	if err != nil {
		return fmt.Errorf("No Access " + err.Error())
	}

	//Iterates through the list returned
	for i, s := range *result.Value {

		//If there is more than one principal throw an error since only one should be get
		if i != 0 {
			return fmt.Errorf("No Access, got more than one principal")
		}

		//The contributor role has a static ID so it checks the RoleDefiniitonID of the service principal by comparing it to
		//an almost static string where only the subscriptionID is different
		if *s.Properties.RoleDefinitionID == "/subscriptions/"+subID+"/providers/Microsoft.Authorization/roleDefinitions/b24988ac-6180-42a0-ab88-20f7382dd24c" {
			fmt.Println("Has access")
			return nil
		}

	}

	//if the result.Value lengt is 0 then the loop never happens and that means the principal was never found
	fmt.Println("Principal not found")
	return nil

}

func returnEnv(appID, tenantID, subID, secretKey string) {

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
