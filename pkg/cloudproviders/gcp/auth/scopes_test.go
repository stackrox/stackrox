package auth

import (
	"testing"

	artifactv1 "cloud.google.com/go/artifactregistry/apiv1"
)

func TestGCPAuthScopesMatchSDK(t *testing.T) {
	// This test will catch if the default scopes ever change in the API.
	url := "https://github.com/googleapis/google-cloud-go/blob/artifactregistry/v1.26.0/artifactregistry/apiv1/helpers.go#L53-L59"

	actual := DefaultAuthScopes()
	expected := artifactv1.DefaultAuthScopes()

	if actual != expected {
		t.Fatalf("Expected API DefaultAuthScopres '%d', to match our copy: '%d'. Check %s and update our copy defaultAuthScopes to match.", expected, actual, url)
	}
}
