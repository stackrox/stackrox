package auth

import (
	"testing"

	"golang.org/x/oauth2/google"
	storagev1 "google.golang.org/api/storage/v1"
)

func TestDefaultGCPCredChainMatchesSDK(t *testing.T) {
	credManager := &defaultCredentialsManager{}
	actual, err := credManager.GetCredentials()
	if err != nil {
		t.Fatalf("Error getting default GCP credentials chain.")
	}

	expected, err := google.FindDefaultCredentials(nil, storagev1.CloudPlatformScope)
	if err != nil {
		t.Fatalf("Error getting default GCP credentials chain from SDK.")
	}

	if actual != expected {
		t.Fatalf("Expected credentials like %v but received %v. Check if default scopes need updated.", expected, actual)
	}
}
