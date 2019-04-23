package deploymentenvs

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
)

// GetDeploymentEnvFromProviderMetadata returns a deployment environment ID string from the given provider
// metadata.
func GetDeploymentEnvFromProviderMetadata(metadata *storage.ProviderMetadata) string {
	envBaseName := getDeploymentEnvBaseNameFromProviderMetadata(metadata)
	if metadata.GetVerified() {
		return envBaseName
	}
	return fmt.Sprintf("~%s", envBaseName)
}

func getDeploymentEnvBaseNameFromProviderMetadata(metadata *storage.ProviderMetadata) string {
	if gcpProject := metadata.GetGoogle().GetProject(); gcpProject != "" {
		return fmt.Sprintf("gcp/%s", gcpProject)
	}
	if awsAccountID := metadata.GetAws().GetAccountId(); awsAccountID != "" {
		return fmt.Sprintf("aws/%s", awsAccountID)
	}
	if azureSubscriptionID := metadata.GetAzure().GetSubscriptionId(); azureSubscriptionID != "" {
		return fmt.Sprintf("azure/%s", azureSubscriptionID)
	}
	return "unknown"
}
