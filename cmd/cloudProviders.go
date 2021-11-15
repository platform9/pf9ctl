package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/config"
	"github.com/platform9/pf9ctl/pkg/objects"
	"github.com/platform9/pf9ctl/pkg/pmk"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
	Run:   checkGoogleProviderRun,
}

var checkAmazonProviderCmd = &cobra.Command{
	Use:   "check-amazon-provider",
	Short: "checks if user has amazon cloud permissions",
	Long:  "Checks if user has the correct permissions to use the amazon cloud provider",
	Run:   checkAmazonProviderRun,
}

var checkAzureProviderCmd = &cobra.Command{
	Use:   "check-azure-provider",
	Short: "checks if user has azure cloud permissions",
	Long:  "Checks if service principal has the correct permissions to use the azure cloud provider",
	Run:   checkAzureProviderRun,
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

	cfg := &objects.Config{}
	var err error
	if cmd.Flags().Changed("no-prompt") {
		flagsNotSet := checkFlags(cmd)
		if len(flagsNotSet) > 0 {
			fmt.Printf(color.Red("x ")+"Missing required flags: %v\n", strings.Join(flagsNotSet, ", "))
			os.Exit(1)
		}
		err = config.LoadConfig("google.json", cfg, objects.NodeConfig{})
	} else {
		err = config.LoadConfigInteractive("google.json", cfg, objects.NodeConfig{})
	}
	if err != nil {
		zap.S().Fatalf("Unable to load the context: %s\n", err.Error())
	}

	if !pmk.CheckGoogleProvider(cfg.GooglePath, cfg.GoogleProjectName, cfg.GoogleServiceEmail) {
		os.Exit(1)
	}

}

func checkAmazonProviderRun(cmd *cobra.Command, args []string) {
	cfg := &objects.Config{}
	var err error
	if cmd.Flags().Changed("no-prompt") {
		flagsNotSet := checkFlags(cmd)
		if len(flagsNotSet) > 0 {
			fmt.Printf(color.Red("x ")+"Missing required flags: %v\n", strings.Join(flagsNotSet, ", "))
			os.Exit(1)
		}
		err = config.LoadConfig("amazon.json", cfg, objects.NodeConfig{})
	} else {
		err = config.LoadConfigInteractive("amazon.json", cfg, objects.NodeConfig{})
	}
	if err != nil {
		zap.S().Fatalf("Unable to load the context: %s\n", err.Error())
	}

	if !pmk.CheckAmazonPovider(cfg.AwsIamUsername, cfg.AwsAccessKey, cfg.AwsSecretKey, cfg.AwsRegion) {
		os.Exit(1)
	}
}

func checkAzureProviderRun(cmd *cobra.Command, args []string) {

	cfg := &objects.Config{}
	var err error
	if cmd.Flags().Changed("no-prompt") {
		flagsNotSet := checkFlags(cmd)
		if len(flagsNotSet) > 0 {
			fmt.Printf(color.Red("x ")+"Missing required flags: %v\n", strings.Join(flagsNotSet, ", "))
			os.Exit(1)
		}
		err = config.LoadConfig("azure.json", cfg, objects.NodeConfig{})
	} else {
		err = config.LoadConfigInteractive("azure.json", cfg, objects.NodeConfig{})
	}
	if err != nil {
		zap.S().Fatalf("Unable to load the context: %s\n")
	}

	if !pmk.CheckAzureProvider(cfg.AzureTetant, cfg.AzureClient, cfg.AzureSubscription, cfg.AzureSecret) {
		os.Exit(1)
	}

}

func checkFlags(cmd *cobra.Command) []string {
	flagsNotSet := []string{}
	fss := cmd.LocalFlags()
	fss.VisitAll(func(f *pflag.Flag) {
		if f.Name != "help" && !f.Changed {
			flagsNotSet = append(flagsNotSet, f.Name)
		}
	})
	return flagsNotSet
}
