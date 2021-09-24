package cmd

import (
	"errors"

	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	awsIamUser         string
	awsAccessKey       string
	awsSecretKey       string
	awsRegion          string
	azureAppID         string
	azureTenantID      string
	azureSubID         string
	azureSecretKey     string
	googleProjectName  string
	googleServiceEmail string
	loadConfig         bool
)

var checkGoogleProviderCmd = &cobra.Command{
	Use:   "check-google-provider [project-name service-account-email]",
	Short: "checks if user has google cloud permisisons",
	Long:  "Checks if service account has the correct roles to use the google cloud provider",
	Args: func(checkGoogleProviderCmd *cobra.Command, args []string) error {
		if len(args) != 0 && len(args) != 2 {
			return errors.New("Only the Project Name and the Service Account Email is needed")
		}

		if len(args) == 2 {
			googleProjectName = args[0]
			googleServiceEmail = args[1]
			loadConfig = false
		} else {
			loadConfig = true
		}

		return nil
	},
	Run: checkGoogleProviderRun,
}

var checkAmazonProviderCmd = &cobra.Command{
	Use:   "check-amazon-provider [iam-user access-key secret-key region]",
	Short: "checks if user has amazon cloud permisisons",
	Long:  "Checks if user has the correct permissions to use the amazon cloud provider",
	Args: func(checkGoogleProviderCmd *cobra.Command, args []string) error {
		if len(args) != 0 && len(args) != 3 && len(args) != 4 {
			return errors.New("Only the IAM user, access key and secret key are needed")
		}

		if len(args) == 3 || len(args) == 4 {
			awsIamUser = args[0]
			awsAccessKey = args[1]
			awsSecretKey = args[2]
			if len(args) == 3 {
				awsRegion = "us-east-1"
			} else {
				awsRegion = args[3]
			}
			loadConfig = false
		} else {
			loadConfig = true
		}

		return nil
	},
	Run: checkAmazonProviderRun,
}

var checkAzureProviderCmd = &cobra.Command{
	Use:   "check-azure-provider",
	Short: "checks if user has azure cloud permisisons",
	Long:  "Checks if service principal has the correct permissions to use the azure cloud provider",
	Args: func(checkGoogleProviderCmd *cobra.Command, args []string) error {
		if len(args) != 0 && len(args) != 4 {
			return errors.New("Only the TenantID, ApplicationID, SubscriptionID and Secret Key are needed")
		}

		if len(args) == 4 {
			azureTenantID = args[0]
			azureAppID = args[1]
			azureSubID = args[2]
			azureSecretKey = args[3]
			loadConfig = false
		} else {
			loadConfig = true
		}

		return nil
	},
	Run: checkAzureProviderRun,
}

func init() {

	rootCmd.AddCommand(checkGoogleProviderCmd)
	rootCmd.AddCommand(checkAmazonProviderCmd)
	rootCmd.AddCommand(checkAzureProviderCmd)
}

func checkGoogleProviderRun(cmd *cobra.Command, args []string) {

	if !loadConfig {
		pmk.CheckGoogleProvider(googleProjectName, googleServiceEmail)
		return
	}

	ctx, err := pmk.LoadConfig("google.json")

	if err != nil {
		zap.S().Fatalf("Unable to load the context: %s\n", err.Error())
	}

	pmk.CheckGoogleProvider(ctx.GoogleProjectName, ctx.GoogleServiceEmail)

}

func checkAmazonProviderRun(cmd *cobra.Command, args []string) {

	if !loadConfig {
		pmk.CheckAmazonPovider(awsIamUser, awsAccessKey, awsSecretKey, awsRegion)
		return
	}

	ctx, err := pmk.LoadConfig("amazon.json")

	if err != nil {
		zap.S().Fatalf("Unable to load the context: %s\n", err.Error())
	}

	pmk.CheckAmazonPovider(ctx.AwsIamUsername, ctx.AwsAccessKey, ctx.AwsSecretKey, ctx.AwsRegion)
}

func checkAzureProviderRun(cmd *cobra.Command, args []string) {

	if !loadConfig {
		pmk.CheckAzureProvider(azureTenantID, azureAppID, azureSubID, azureSecretKey)
		return
	}

	ctx, err := pmk.LoadConfig("azure.json")

	if err != nil {
		zap.S().Fatalf("Unable to load the context: %s\n")
	}

	pmk.CheckAzureProvider(ctx.AzureTetant, ctx.AzureApplication, ctx.AzureSubscription, ctx.AzureSecret)

}
