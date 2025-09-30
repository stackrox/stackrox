package credentials

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// Credentials centralizes all external service credentials for E2E testing
type Credentials struct {
	// Core StackRox Authentication
	ROXUsername     string
	ROXAdminPassword string
	APIHostname     string
	APIPort         string

	// Container Registries (Real Image Scanning)
	RegistryUsername string
	RegistryPassword string

	// Google Container Registry
	GoogleGCRCredentials string

	// Red Hat Registry
	RedHatUsername string
	RedHatPassword string

	// Azure Container Registry
	AzureRegistryPassword string

	// Cloud Storage (Real Backup Operations)
	AWSAccessKeyID     string
	AWSSecretAccessKey string
	AWSS3BucketName    string
	AWSS3BucketRegion  string

	// Google Cloud Storage
	GCPServiceAccount string
	GCSBucketName     string

	// Azure Storage
	AzureClientID     string
	AzureClientSecret string
	AzureTenantID     string

	// Notification Services (Real Message Delivery)
	SlackWebhookURL    string
	SlackAltWebhook    string
	GenericWebhookServerCA string

	// OpenShift/Cloud Sources Integration
	OCMOfflineToken            string
	CloudSourcesTestOCMClientID     string
	CloudSourcesTestOCMClientSecret string

	// Infrastructure
	KubeConfig string
	Cluster    string

	// Test Environment Mode
	TestEnv string
}

// Load reads credentials from environment variables and validates required ones
func Load() (*Credentials, error) {
	creds := &Credentials{
		// Core StackRox Authentication
		ROXUsername:     getEnvOrDefault("ROX_USERNAME", "admin"),
		ROXAdminPassword: os.Getenv("ROX_ADMIN_PASSWORD"),
		APIHostname:     getEnvOrDefault("API_HOSTNAME", "localhost"),
		APIPort:         getEnvOrDefault("API_PORT", "8000"),

		// Container Registries
		RegistryUsername: os.Getenv("REGISTRY_USERNAME"),
		RegistryPassword: os.Getenv("REGISTRY_PASSWORD"),
		GoogleGCRCredentials: os.Getenv("GOOGLE_CREDENTIALS_GCR_SCANNER_V2"),
		RedHatUsername:   os.Getenv("REDHAT_USERNAME"),
		RedHatPassword:   os.Getenv("REDHAT_PASSWORD"),
		AzureRegistryPassword: os.Getenv("AZURE_REGISTRY_PASSWORD"),

		// Cloud Storage
		AWSAccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
		AWSSecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		AWSS3BucketName:    os.Getenv("AWS_S3_BUCKET_NAME"),
		AWSS3BucketRegion:  os.Getenv("AWS_S3_BUCKET_REGION"),
		GCPServiceAccount:  os.Getenv("GCP_SERVICE_ACCOUNT"),
		GCSBucketName:      os.Getenv("GCS_BUCKET_NAME"),
		AzureClientID:      os.Getenv("AZURE_CLIENT_ID"),
		AzureClientSecret:  os.Getenv("AZURE_CLIENT_SECRET"),
		AzureTenantID:      os.Getenv("AZURE_TENANT_ID"),

		// Notification Services
		SlackWebhookURL:    os.Getenv("SLACK_WEBHOOK_URL"),
		SlackAltWebhook:    os.Getenv("SLACK_ALT_WEBHOOK"),
		GenericWebhookServerCA: os.Getenv("GENERIC_WEBHOOK_SERVER_CA_CONTENTS"),

		// OpenShift/Cloud Sources
		OCMOfflineToken:            os.Getenv("OCM_OFFLINE_TOKEN"),
		CloudSourcesTestOCMClientID:     os.Getenv("CLOUD_SOURCES_TEST_OCM_CLIENT_ID"),
		CloudSourcesTestOCMClientSecret: os.Getenv("CLOUD_SOURCES_TEST_OCM_CLIENT_SECRET"),

		// Infrastructure
		KubeConfig: os.Getenv("KUBECONFIG"),
		Cluster:    os.Getenv("CLUSTER"),

		// Test Environment
		TestEnv: getEnvOrDefault("ROX_TEST_ENV", "development"),
	}

	// Validate required credentials based on test environment
	if err := creds.validate(); err != nil {
		return nil, err
	}

	return creds, nil
}

// validate checks required credentials based on test environment
func (c *Credentials) validate() error {
	var missingCreds []string

	// Always require basic StackRox authentication
	if c.ROXAdminPassword == "" {
		missingCreds = append(missingCreds, "ROX_ADMIN_PASSWORD")
	}

	// For master/nightly CI, require external service credentials
	if c.TestEnv == "ci-master" {
		requiredForMaster := map[string]string{
			"AWS_ACCESS_KEY_ID":                   c.AWSAccessKeyID,
			"GOOGLE_CREDENTIALS_GCR_SCANNER_V2":   c.GoogleGCRCredentials,
			"SLACK_WEBHOOK_URL":                   c.SlackWebhookURL,
			"REGISTRY_USERNAME":                   c.RegistryUsername,
			"REGISTRY_PASSWORD":                   c.RegistryPassword,
		}

		for envVar, value := range requiredForMaster {
			if value == "" {
				missingCreds = append(missingCreds, envVar)
			}
		}
	}

	if len(missingCreds) > 0 {
		return fmt.Errorf("missing required credentials for test environment '%s': %s",
			c.TestEnv, strings.Join(missingCreds, ", "))
	}

	return nil
}

// Helper methods for checking credential availability

// HasGCRCredentials returns true if Google Container Registry credentials are available
func (c *Credentials) HasGCRCredentials() bool {
	return c.GoogleGCRCredentials != ""
}

// HasAWSCredentials returns true if AWS credentials are available
func (c *Credentials) HasAWSCredentials() bool {
	return c.AWSAccessKeyID != "" && c.AWSSecretAccessKey != ""
}

// HasAzureCredentials returns true if Azure credentials are available
func (c *Credentials) HasAzureCredentials() bool {
	return c.AzureClientID != "" && c.AzureClientSecret != "" && c.AzureTenantID != ""
}

// HasRegistryCredentials returns true if container registry credentials are available
func (c *Credentials) HasRegistryCredentials() bool {
	return c.RegistryUsername != "" && c.RegistryPassword != ""
}

// HasSlackCredentials returns true if Slack webhook URL is available
func (c *Credentials) HasSlackCredentials() bool {
	return c.SlackWebhookURL != ""
}

// HasGCSCredentials returns true if Google Cloud Storage credentials are available
func (c *Credentials) HasGCSCredentials() bool {
	return c.GCPServiceAccount != "" && c.GCSBucketName != ""
}

// HasRedHatCredentials returns true if Red Hat registry credentials are available
func (c *Credentials) HasRedHatCredentials() bool {
	return c.RedHatUsername != "" && c.RedHatPassword != ""
}

// IsDevelopmentMode returns true if running in development mode
func (c *Credentials) IsDevelopmentMode() bool {
	return c.TestEnv == "development"
}

// IsCIPRMode returns true if running in CI PR mode (mocked external services)
func (c *Credentials) IsCIPRMode() bool {
	return c.TestEnv == "ci-pr"
}

// IsCIMasterMode returns true if running in CI master mode (real external services)
func (c *Credentials) IsCIMasterMode() bool {
	return c.TestEnv == "ci-master"
}

// ShouldUseMockServices returns true if external services should be mocked
func (c *Credentials) ShouldUseMockServices() bool {
	return c.IsDevelopmentMode() || c.IsCIPRMode()
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// NewClientError creates a standardized error for missing credentials
func NewClientError(serviceName string, err error) error {
	if err.Error() == "credentials required" {
		return fmt.Errorf("%s not configured: %s. Set appropriate environment variables or run in development mode", serviceName, err.Error())
	}
	return fmt.Errorf("%s client error: %w", serviceName, err)
}