package env

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/pkg/env"
)

var (
	// Core StackRox Authentication - reuse existing settings where available
	ROXUsername      = env.RegisterSetting("ROX_USERNAME", env.WithDefault("admin"))
	ROXAdminPassword = env.PasswordEnv // Already registered in pkg/env/roxctl.go
	APIHostname      = env.RegisterSetting("API_HOSTNAME", env.WithDefault("localhost"))
	APIPort          = env.RegisterSetting("API_PORT", env.WithDefault("8000"))

	// Container Registries (Real Image Scanning)
	RegistryUsername = env.RegisterSetting("REGISTRY_USERNAME")
	RegistryPassword = env.RegisterSetting("REGISTRY_PASSWORD")

	// Google Container Registry
	GoogleGCRCredentials = env.RegisterSetting("GOOGLE_CREDENTIALS_GCR_SCANNER_V2")

	// Red Hat Registry
	RedHatUsername = env.RegisterSetting("REDHAT_USERNAME")
	RedHatPassword = env.RegisterSetting("REDHAT_PASSWORD")

	// Azure Container Registry
	AzureRegistryPassword = env.RegisterSetting("AZURE_REGISTRY_PASSWORD")

	// Cloud Storage (Real Backup Operations)
	AWSAccessKeyID     = env.RegisterSetting("AWS_ACCESS_KEY_ID")
	AWSSecretAccessKey = env.RegisterSetting("AWS_SECRET_ACCESS_KEY")
	AWSS3BucketName    = env.RegisterSetting("AWS_S3_BUCKET_NAME")
	AWSS3BucketRegion  = env.RegisterSetting("AWS_S3_BUCKET_REGION")

	// Google Cloud Storage
	GCPServiceAccount = env.RegisterSetting("GCP_SERVICE_ACCOUNT")
	GCSBucketName     = env.RegisterSetting("GCS_BUCKET_NAME")

	// Azure Storage
	AzureClientID     = env.RegisterSetting("AZURE_CLIENT_ID")
	AzureClientSecret = env.RegisterSetting("AZURE_CLIENT_SECRET")
	AzureTenantID     = env.RegisterSetting("AZURE_TENANT_ID")

	// Notification Services (Real Message Delivery)
	SlackWebhookURL        = env.RegisterSetting("SLACK_WEBHOOK_URL")
	SlackAltWebhook        = env.RegisterSetting("SLACK_ALT_WEBHOOK")
	GenericWebhookServerCA = env.RegisterSetting("GENERIC_WEBHOOK_SERVER_CA_CONTENTS")

	// OpenShift/Cloud Sources Integration
	OCMOfflineToken                 = env.RegisterSetting("OCM_OFFLINE_TOKEN")
	CloudSourcesTestOCMClientID     = env.RegisterSetting("CLOUD_SOURCES_TEST_OCM_CLIENT_ID")
	CloudSourcesTestOCMClientSecret = env.RegisterSetting("CLOUD_SOURCES_TEST_OCM_CLIENT_SECRET")

	// Infrastructure
	KubeConfig = env.RegisterSetting("KUBECONFIG")
	Cluster    = env.RegisterSetting("CLUSTER")

	// Test Environment Mode
	TestEnv = env.RegisterSetting("ROX_TEST_ENV", env.WithDefault("development"))
)

// Validate checks required credentials based on test environment
func Validate() error {
	var missingCreds []string

	// Always require basic StackRox authentication
	if ROXAdminPassword.Setting() == "" {
		missingCreds = append(missingCreds, ROXAdminPassword.EnvVar())
	}

	// For master/nightly CI, require external service credentials
	if TestEnv.Setting() == "ci-master" {
		requiredForMaster := []env.Setting{
			AWSAccessKeyID,
			GoogleGCRCredentials,
			SlackWebhookURL,
			RegistryUsername,
			RegistryPassword,
		}

		for _, setting := range requiredForMaster {
			if setting.Setting() == "" {
				missingCreds = append(missingCreds, setting.EnvVar())
			}
		}
	}

	if len(missingCreds) > 0 {
		return fmt.Errorf("missing required credentials for test environment '%s': %s",
			TestEnv.Setting(), strings.Join(missingCreds, ", "))
	}

	return nil
}

// Helper functions for checking credential availability

// HasGCRCredentials returns true if Google Container Registry credentials are available
func HasGCRCredentials() bool {
	return GoogleGCRCredentials.Setting() != ""
}

// HasAWSCredentials returns true if AWS credentials are available
func HasAWSCredentials() bool {
	return AWSAccessKeyID.Setting() != "" && AWSSecretAccessKey.Setting() != ""
}

// HasAzureCredentials returns true if Azure credentials are available
func HasAzureCredentials() bool {
	return AzureClientID.Setting() != "" && AzureClientSecret.Setting() != "" && AzureTenantID.Setting() != ""
}

// HasRegistryCredentials returns true if container registry credentials are available
func HasRegistryCredentials() bool {
	return RegistryUsername.Setting() != "" && RegistryPassword.Setting() != ""
}

// HasSlackCredentials returns true if Slack webhook URL is available
func HasSlackCredentials() bool {
	return SlackWebhookURL.Setting() != ""
}

// HasGCSCredentials returns true if Google Cloud Storage credentials are available
func HasGCSCredentials() bool {
	return GCPServiceAccount.Setting() != "" && GCSBucketName.Setting() != ""
}

// HasRedHatCredentials returns true if Red Hat registry credentials are available
func HasRedHatCredentials() bool {
	return RedHatUsername.Setting() != "" && RedHatPassword.Setting() != ""
}

// IsDevelopmentMode returns true if running in development mode
func IsDevelopmentMode() bool {
	return TestEnv.Setting() == "development"
}

// IsCIPRMode returns true if running in CI PR mode (mocked external services)
func IsCIPRMode() bool {
	return TestEnv.Setting() == "ci-pr"
}

// IsCIMasterMode returns true if running in CI master mode (real external services)
func IsCIMasterMode() bool {
	return TestEnv.Setting() == "ci-master"
}

// ShouldUseMockServices returns true if external services should be mocked
func ShouldUseMockServices() bool {
	return IsDevelopmentMode() || IsCIPRMode()
}

// NewClientError creates a standardized error for missing credentials
func NewClientError(serviceName string, err error) error {
	if err.Error() == "credentials required" {
		return fmt.Errorf("%s not configured: %s. Set appropriate environment variables or run in development mode", serviceName, err.Error())
	}
	return fmt.Errorf("%s client error: %w", serviceName, err)
}
