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
	loadConfig bool
)

var checkGoogleProviderCmd = &cobra.Command{
	Use:   "check-google-provider",
	Short: "Checks if user has google cloud permissions",
	Long:  "Checks if service account has the correct roles to use the google cloud provider",
	Run:   checkGoogleProviderRun,
}

var checkAmazonProviderCmd = &cobra.Command{
	Use:   "check-amazon-provider",
	Short: "Checks if user has amazon cloud permissions",
	Long:  "Checks if user has the correct permissions to use the amazon cloud provider",
	Run:   checkAmazonProviderRun,
}

var checkAzureProviderCmd = &cobra.Command{
	Use:   "check-azure-provider",
	Short: "Checks if user has azure cloud permissions",
	Long:  "Checks if service principal has the correct permissions to use the azure cloud provider",
	Run:   checkAzureProviderRun,
}

func init() {

	checkGoogleProviderCmd.Flags().StringVarP(&cfg.GooglePath, "service-account-path", "p", "", "sets the service account path (required)")
	checkGoogleProviderCmd.Flags().StringVarP(&cfg.GoogleProjectName, "project-name", "n", "", "sets the project name (required)")
	checkGoogleProviderCmd.Flags().StringVarP(&cfg.GoogleServiceEmail, "service-account-email", "e", "", "sets the service account email (required)")
	rootCmd.AddCommand(checkGoogleProviderCmd)

	checkAmazonProviderCmd.Flags().StringVarP(&cfg.AwsIamUsername, "iam-user", "i", "", "sets the iam user (required)")
	checkAmazonProviderCmd.Flags().StringVarP(&cfg.AwsAccessKey, "access-key", "a", "", "sets the access key (required)")
	checkAmazonProviderCmd.Flags().StringVarP(&cfg.AwsSecretKey, "secret-key", "s", "", "sets the secret key (required)")
	checkAmazonProviderCmd.Flags().StringVarP(&cfg.AwsRegion, "region", "r", "", "sets the region (required)")
	rootCmd.AddCommand(checkAmazonProviderCmd)

	checkAzureProviderCmd.Flags().StringVarP(&cfg.AzureTenant, "tenant-id", "t", "", "sets the tenant id (required)")
	checkAzureProviderCmd.Flags().StringVarP(&cfg.AzureClient, "client-id", "c", "", "sets the client(applicaiton) id (required)")
	checkAzureProviderCmd.Flags().StringVarP(&cfg.AzureSubscription, "subscription-id", "s", "", "sets the ssubscription id (required)")
	checkAzureProviderCmd.Flags().StringVarP(&cfg.AzureSecret, "secret-key", "k", "", "sets the secret key (required)")
	rootCmd.AddCommand(checkAzureProviderCmd)
}

func checkGoogleProviderRun(cmd *cobra.Command, args []string) {

	var err error
	if cmd.Flags().Changed("no-prompt") {
		flagsNotSet := checkFlags(cmd)
		if len(flagsNotSet) > 0 {
			fmt.Printf(color.Red("x ")+"Missing required flags: %v\n", strings.Join(flagsNotSet, ", "))
			os.Exit(1)
		}
	} else {
		err = config.GetConfigRecursive("google.json", &cfg, objects.NodeConfig{})
	}
	if err != nil {
		zap.S().Fatalf("Unable to load the context: %s\n", err.Error())
	}

	if !pmk.CheckGoogleProvider(cfg.GooglePath, cfg.GoogleProjectName, cfg.GoogleServiceEmail) {
		os.Exit(1)
	}

}

func checkAmazonProviderRun(cmd *cobra.Command, args []string) {
	var err error
	if cmd.Flags().Changed("no-prompt") {
		flagsNotSet := checkFlags(cmd)
		if len(flagsNotSet) > 0 {
			fmt.Printf(color.Red("x ")+"Missing required flags: %v\n", strings.Join(flagsNotSet, ", "))
			os.Exit(1)
		}
	} else {
		err = config.GetConfigRecursive("amazon.json", &cfg, objects.NodeConfig{})
	}
	if err != nil {
		zap.S().Fatalf("Unable to load the context: %s\n", err.Error())
	}
	if !pmk.CheckAmazonPovider(cfg.AwsIamUsername, cfg.AwsAccessKey, cfg.AwsSecretKey, cfg.AwsRegion) {
		os.Exit(1)
	}
}

func checkAzureProviderRun(cmd *cobra.Command, args []string) {

	var err error
	if cmd.Flags().Changed("no-prompt") {
		flagsNotSet := checkFlags(cmd)
		if len(flagsNotSet) > 0 {
			fmt.Printf(color.Red("x ")+"Missing required flags: %v\n", strings.Join(flagsNotSet, ", "))
			os.Exit(1)
		}
	} else {
		err = config.GetConfigRecursive("azure.json", &cfg, objects.NodeConfig{})
	}
	if err != nil {
		zap.S().Fatalf("Unable to load the context: %s\n")
	}

	if !pmk.CheckAzureProvider(cfg.AzureTenant, cfg.AzureClient, cfg.AzureSubscription, cfg.AzureSecret) {
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
