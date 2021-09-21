package cmd

import (
	"errors"

	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	path           string
	awsIamUser     string
	awsKeyID       string
	awsSecretKey   string
	awsRegion      string
	azureAppID     string
	azureTenantID  string
	azureSubID     string
	azureSecretKey string
)

var checkGoogleProviderCmd = &cobra.Command{
	Use:   "check-google-provider path-to-json",
	Short: "checks if user has google cloud permisisons",
	Long:  "Checks if service principle json has the correct permissions to use the google cloud provider",
	Args: func(checkGoogleProviderCmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return errors.New("Only path to service principle json is required")
		} else if len(args) < 1 {
			return errors.New("Path to service principle json is required")
		}
		path = args[0]
		return nil
	},
	Run: checkGoogleProviderRun,
}

var checkAmazonProviderCmd = &cobra.Command{
	Use:   "check-amazon-provider",
	Short: "checks if user has amazon cloud permisisons",
	Long:  "Checks if user has the correct permissions to use the amazon cloud provider",
	Args: func(checkGoogleProviderCmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errors.New("No parameters are required.")
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
		if len(args) > 0 {
			return errors.New("No parameters are required.")
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

	pmk.CheckGoogleProvider(path)

}

func checkAmazonProviderRun(cmd *cobra.Command, args []string) {

	ctx, err := pmk.LoadConfig("amazon.json")

	if err != nil {
		zap.S().Fatalf("Unable to load the context: %s\n", err.Error())
	}

	pmk.CheckAmazonPovider(ctx.AwsIamUsername, ctx.AwsAccessKey, ctx.AwsSecretKey, ctx.AwsRegion)
}

func checkAzureProviderRun(cmd *cobra.Command, args []string) {

	ctx, err := pmk.LoadConfig("azure.json")

	if err != nil {
		zap.S().Fatalf("Unable to load the context: %s\n")
	}

	pmk.CheckAzureProvider(ctx.AzureTetant, ctx.AzureApplication, ctx.AzureSubscription, ctx.AzureSecret)

}
