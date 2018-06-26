package quay

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

const (
	// This is a robot token that can only pull from quay.io/integration/nginx
	testOauthToken = "0j9dhT9jCNFpsVAzwLavnyeEy2HWnrfTQnbJgQF8"
)

func TestQuay(t *testing.T) {
	integration := &v1.ImageIntegration{
		IntegrationConfig: &v1.ImageIntegration_Quay{
			Quay: &v1.QuayConfig{
				OauthToken: testOauthToken,
				Endpoint:   "quay.io",
			},
		},
	}

	q, err := newRegistry(integration)
	assert.NoError(t, err)
	assert.NoError(t, q.Test())
}
