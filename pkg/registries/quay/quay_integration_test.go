//go:build integration

package quay

import (
	"strings"
	"testing"

	"github.com/stackrox/rox/generated/storage"
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
	assert.NoError(t, filterOkErrors(q.Test()))
}

func filterOkErrors(err error) error {
	if err != nil &&
		(strings.Contains(err.Error(), "EOF") ||
			strings.Contains(err.Error(), "status=502")) {
		// Ignore failures that can indicate quay.io outage
		return nil
	}
	return err
}
