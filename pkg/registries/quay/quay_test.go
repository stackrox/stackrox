package quay

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

const (
	// This is a robot token that can only pull from quay.io/integration/nginx
	testOauthToken = "0j9dhT9jCNFpsVAzwLavnyeEy2HWnrfTQnbJgQF8"
)

func TestQuay(t *testing.T) {
	integration := &storage.ImageIntegration{
		IntegrationConfig: &storage.ImageIntegration_Quay{
			Quay: &storage.QuayConfig{
				OauthToken: testOauthToken,
				Endpoint:   "quay.io",
			},
		},
	}

	q, err := newRegistry(integration)
	assert.NoError(t, err)
	assert.NoError(t, q.Test())
}
