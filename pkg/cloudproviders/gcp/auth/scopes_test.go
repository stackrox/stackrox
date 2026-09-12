package auth

import (
	"testing"

	artifactv1 "cloud.google.com/go/artifactregistry/apiv1"
	securitycenterv1 "cloud.google.com/go/securitycenter/apiv1"
	storagev1 "google.golang.org/api/storage/v1"
)

func TestGCPAuthScopesMatchSDK(t *testing.T) {
	// This test will catch if the default scopes ever change in the API.
	artifactregistryUrl := "https://pkg.go.dev/cloud.google.com/go/artifactregistry/apiv1#DefaultAuthScopes"
	securitycenterUrl := "https://pkg.go.dev/cloud.google.com/go/securitycenter/apiv1#DefaultAuthScopes"
	storageUrl := "https://pkg.go.dev/google.golang.org/api/storage/v1#pkg-constants"

	actual := DefaultAuthScopes()
	found := make(map[string]bool)
	for _, item := range actual {
		found[item] = true
	}

	expected := artifactv1.DefaultAuthScopes()
	for _, item := range expected {
		if !found[item] {
			t.Fatalf("Expected GCP artifactregistry API DefaultAuthScopes item '%s', to be in our copy: '%v'. Check %s and update our defaultAuthScopes.", item, actual, artifactregistryUrl)
		}
	}

	expected = securitycenterv1.DefaultAuthScopes()
	for _, item := range expected {
		if !found[item] {
			t.Fatalf("Expected GCP securitycenter API DefaultAuthScopes item '%s', to be in our copy: '%v'. Check %s and update our defaultAuthScopes.", item, actual, securitycenterUrl)
		}
	}

	item := storagev1.CloudPlatformScope
	if !found[item] {
		t.Fatalf("Expected GCP storage API CloudPlatformScope '%s', to be in our copy: '%v'. Check %s and update our defaultAuthScopes.", item, actual, storageUrl)
	}
}
