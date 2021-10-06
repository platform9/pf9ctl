package cmd

import (
	"os"

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
	Short: "checks if user has google cloud permissions",
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
	Short: "checks if user has amazon cloud permissions",
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
	Short: "checks if user has azure cloud permissions",
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

	checkGoogleProviderCmd.Flags().StringVarP(&googlePath, "service_account_path", "p", "", "sets the service account path (required)")
	checkGoogleProviderCmd.Flags().StringVarP(&googleProjectName, "project_name", "n", "", "sets the project name (required)")
	checkGoogleProviderCmd.Flags().StringVarP(&googleServiceEmail, "service_account_email", "e", "", "sets the service account email (required)")
	rootCmd.AddCommand(checkGoogleProviderCmd)

	checkAmazonProviderCmd.Flags().StringVarP(&awsIamUser, "iam_user", "i", "", "sets the iam user (required)")
	checkAmazonProviderCmd.Flags().StringVarP(&awsAccessKey, "access_key", "a", "", "sets the access key (required)")
	checkAmazonProviderCmd.Flags().StringVarP(&awsSecretKey, "secret_key", "s", "", "sets the secret key (required)")
	checkAmazonProviderCmd.Flags().StringVarP(&awsRegion, "region", "r", "us-east-1", "sets the region")
	rootCmd.AddCommand(checkAmazonProviderCmd)

	checkAzureProviderCmd.Flags().StringVarP(&azureTenantID, "tenant_id", "t", "", "sets the tenant id (required)")
	checkAzureProviderCmd.Flags().StringVarP(&azureAppID, "client_id", "c", "", "sets the client(applicaiton) id (required)")
	checkAzureProviderCmd.Flags().StringVarP(&azureSubID, "subscription_id", "s", "", "sets the ssubscription id (required)")
	checkAzureProviderCmd.Flags().StringVarP(&azureSecretKey, "secret_key", "k", "", "sets the secret key (required)")
	rootCmd.AddCommand(checkAzureProviderCmd)
}

func checkGoogleProviderRun(cmd *cobra.Command, args []string) {

	if !loadConfig {
		if !pmk.CheckGoogleProvider(googlePath, googleProjectName, googleServiceEmail) {
			os.Exit(1)
		}
		os.Exit(0)
	}

	ctx, err := pmk.LoadConfig("google.json")

	if err != nil {
		zap.S().Fatalf("Unable to load the context: %s\n", err.Error())
	}

	if !pmk.CheckGoogleProvider(ctx.GooglePath, ctx.GoogleProjectName, ctx.GoogleServiceEmail) {
		os.Exit(1)
	}

}

func checkAmazonProviderRun(cmd *cobra.Command, args []string) {

	if !loadConfig {
		if !pmk.CheckAmazonPovider(awsIamUser, awsAccessKey, awsSecretKey, awsRegion) {
			os.Exit(1)
		}
		os.Exit(0)
	}

	ctx, err := pmk.LoadConfig("amazon.json")

	if err != nil {
		zap.S().Fatalf("Unable to load the context: %s\n", err.Error())
	}

	if !pmk.CheckAmazonPovider(ctx.AwsIamUsername, ctx.AwsAccessKey, ctx.AwsSecretKey, ctx.AwsRegion) {
		os.Exit(1)
	}
}

func checkAzureProviderRun(cmd *cobra.Command, args []string) {

	if !loadConfig {
		if !pmk.CheckAzureProvider(azureTenantID, azureAppID, azureSubID, azureSecretKey) {
			os.Exit(1)
		}
		os.Exit(0)
	}

	ctx, err := pmk.LoadConfig("azure.json")

	if err != nil {
		zap.S().Fatalf("Unable to load the context: %s\n")
	}

	if !pmk.CheckAzureProvider(ctx.AzureTetant, ctx.AzureClient, ctx.AzureSubscription, ctx.AzureSecret) {
		os.Exit(1)
	}

}
