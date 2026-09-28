//go:build integration

package quay

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stretchr/testify/assert"
)

const (
	// This is a robot token that can only pull from quay.io/integration/nginx
	testOauthToken = "0j9dhT9jCNFpsVAzwLavnyeEy2HWnrfTQnbJgQF8" //#nosec G101
)

func TestQuay(t *testing.T) {
	t.Setenv("ROX_REGISTRY_RESPONSE_TIMEOUT", "90s")
	t.Setenv("ROX_REGISTRY_CLIENT_TIMEOUT", "120s")

	integration := &storage.ImageIntegration{
		IntegrationConfig: &storage.ImageIntegration_Quay{
			Quay: &storage.QuayConfig{
				OauthToken: testOauthToken,
				Endpoint:   "quay.io",
			},
		},
	}

	q, err := newRegistry(integration, false, nil)
	assert.NoError(t, err)
	err = retry.WithRetry(func() error {
		return q.Test()
	}, retry.Tries(3),
		retry.WithExponentialBackoff(),
		retry.OnFailedAttempts(func(err error) {
			t.Logf("retrying: %v", err)
		}),
	)
	assert.NoError(t, err)
}
