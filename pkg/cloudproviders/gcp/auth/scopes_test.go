package auth

import (
	"testing"

	artifactv1 "cloud.google.com/go/artifactregistry/apiv1"
	"github.com/stretchr/testify/assert"
)

func TestGCPAuthScopesMatchSDK(t *testing.T) {
	assert.ElementsMatch(t, GCPAuthScopes(), artifactv1.DefaultAuthScopes())
}
