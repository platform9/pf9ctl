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
	Use:   "check-amazon-provider IAM-User access-key-id secret-access-key region",
	Short: "checks if user has amazon cloud permisisons",
	Long:  "Checks if user has the correct permissions to use the amazon cloud provider",
	Args: func(checkGoogleProviderCmd *cobra.Command, args []string) error {
		if len(args) > 4 {
			return errors.New("Only IAM USer, keyID, secretKey and region are required")
		} else if len(args) < 3 {
			return errors.New("Not all required fields were sent.")
		}
		awsIamUser = args[0]
		awsKeyID = args[1]
		awsSecretKey = args[2]

		//if the region is sent it will set it
		if len(args) == 4 {
			awsRegion = args[3]
		} else { //if it is not set it will then use aws default region
			awsRegion = "us-east-1"
		}
		return nil
	},
	Run: checkAmazonProviderRun,
}

var checkAzureProviderCmd = &cobra.Command{
	Use:   "check-azure-provider tenantID applicationID subscriptionID secretKey",
	Short: "checks if user has azure cloud permisisons",
	Long:  "Checks if service principal has the correct permissions to use the azure cloud provider",
	Args: func(checkGoogleProviderCmd *cobra.Command, args []string) error {
		if len(args) > 4 {
			return errors.New("Only tenantID, applicationID, subscriptioID and secretKey are required")
		} else if len(args) < 4 {
			return errors.New("Not all required fields were sent.")
		}
		azureTenantID = args[0]
		azureAppID = args[1]
		azureSubID = args[2]
		azureSecretKey = args[3]
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

	if err := pmk.CheckGoogleProvider(path); err != nil {
		zap.S().Fatalf("Unable to verify google provider ", err)
		return
	}

}

func checkAmazonProviderRun(cmd *cobra.Command, args []string) {

	if err := pmk.CheckAmazonPovider(awsIamUser, awsKeyID, awsSecretKey, awsRegion); err != nil {
		zap.S().Fatalf("Unable to verify amazon provider ", err)
		return
	}

}

func checkAzureProviderRun(cmd *cobra.Command, args []string) {

	if err := pmk.CheckAzureProvider(azureTenantID, azureAppID, azureSubID, azureSecretKey); err != nil {
		zap.S().Fatalf("Unable to verify azure provider ", err)
		return
	}

}
