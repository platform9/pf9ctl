package cmd

import (
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
	googlePath         string
	googleProjectName  string
	googleServiceEmail string
	loadConfig         bool
)

var checkGoogleProviderCmd = &cobra.Command{
	Use:   "check-google-provider",
	Short: "checks if user has google cloud permisisons",
	Long:  "Checks if service account has the correct roles to use the google cloud provider",
	Args: func(checkGoogleProviderCmd *cobra.Command, args []string) error {

		if checkGoogleProviderCmd.Flags().Changed("service_account_path") || checkGoogleProviderCmd.Flags().Changed("project_name") || checkGoogleProviderCmd.Flags().Changed("service_account_email") {
			loadConfig = false
		} else {
			loadConfig = true
		}

		return nil
	},
	Run: checkGoogleProviderRun,
}

var checkAmazonProviderCmd = &cobra.Command{
	Use:   "check-amazon-provider",
	Short: "checks if user has amazon cloud permisisons",
	Long:  "Checks if user has the correct permissions to use the amazon cloud provider",
	Args: func(checkAmazonProviderCmd *cobra.Command, args []string) error {
		if checkAmazonProviderCmd.Flags().Changed("iam_user") || checkAmazonProviderCmd.Flags().Changed("access_key") || checkAmazonProviderCmd.Flags().Changed("secret_key") {
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
	Args: func(checkAzureProviderCmd *cobra.Command, args []string) error {
		if checkAzureProviderCmd.Flags().Changed("tenant_id") || checkAzureProviderCmd.Flags().Changed("client_id") || checkAzureProviderCmd.Flags().Changed("subscription_id") || checkAzureProviderCmd.Flags().Changed("secret_key") {
			loadConfig = false
		} else {
			loadConfig = true
		}

		return nil
	},
	Run: checkAzureProviderRun,
}

func init() {

	checkGoogleProviderCmd.Flags().StringVarP(&googlePath, "service_account_path", "p", "", "sets the service account path")
	checkGoogleProviderCmd.Flags().StringVarP(&googleProjectName, "project_name", "n", "", "sets the project name")
	checkGoogleProviderCmd.Flags().StringVarP(&googleServiceEmail, "service_account_email", "e", "", "sets the service account email")
	rootCmd.AddCommand(checkGoogleProviderCmd)

	checkAmazonProviderCmd.Flags().StringVarP(&awsIamUser, "iam_user", "i", "", "sets the iam user")
	checkAmazonProviderCmd.Flags().StringVarP(&awsAccessKey, "access_key", "a", "", "sets the access key")
	checkAmazonProviderCmd.Flags().StringVarP(&awsSecretKey, "secret_key", "s", "", "sets the secret key")
	checkAmazonProviderCmd.Flags().StringVarP(&awsRegion, "region", "r", "us-east-1", "sets the region")
	rootCmd.AddCommand(checkAmazonProviderCmd)

	checkAzureProviderCmd.Flags().StringVarP(&azureTenantID, "tenant_id", "t", "", "sets the tenant id")
	checkAzureProviderCmd.Flags().StringVarP(&azureAppID, "client_id", "c", "", "sets the client(applicaiton) id")
	checkAzureProviderCmd.Flags().StringVarP(&azureSubID, "subscription_id", "s", "", "sets the ssubscription id")
	checkAzureProviderCmd.Flags().StringVarP(&azureSecretKey, "secret_key", "k", "", "sets the secret key")
	rootCmd.AddCommand(checkAzureProviderCmd)
}

func checkGoogleProviderRun(cmd *cobra.Command, args []string) {

	if !loadConfig {
		pmk.CheckGoogleProvider(googlePath, googleProjectName, googleServiceEmail)
		return
	}

	ctx, err := pmk.LoadConfig("google.json")

	if err != nil {
		zap.S().Fatalf("Unable to load the context: %s\n", err.Error())
	}

	pmk.CheckGoogleProvider(ctx.GooglePath, ctx.GoogleProjectName, ctx.GoogleServiceEmail)

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

	pmk.CheckAzureProvider(ctx.AzureTetant, ctx.AzureClient, ctx.AzureSubscription, ctx.AzureSecret)

}
