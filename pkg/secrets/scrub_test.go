package secrets

import (
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestScrubSecretsFromMap(t *testing.T) {
	m := map[string]string{
		// Don't scrub this:
		"endpoint": "endpoint",

		// Scrub all of these:
		"oauthToken":     "token",
		"oauthtoken":     "token",
		"password":       "password",
		"secretKey":      "secret!",
		"secretkey":      "secret",
		"serviceAccount": "sa",
		"serviceaccount": "sa",
	}
	assert.Equal(t, map[string]string{"endpoint": "endpoint"}, ScrubSecretsFromMap(m))
}

type config struct {
	OauthToken string
}

type toplevel struct {
	Name     string
	Password string
	Config   *config
}

func TestScrubSecretsFromStruct(t *testing.T) {
	testStruct := &toplevel{Name: "name", Password: "password"}
	ScrubSecretsFromStruct(testStruct)
	assert.Empty(t, testStruct.Password)
	assert.Equal(t, testStruct.Name, "name")
}

func TestScrubFromNestedStruct(t *testing.T) {
	testStruct := &toplevel{
		Name:     "name",
		Password: "password",
		Config: &config{
			OauthToken: "oauth",
		},
	}
	ScrubSecretsFromStruct(testStruct)
	assert.Empty(t, testStruct.Password)
	assert.Equal(t, "name", testStruct.Name)
	assert.Equal(t, "", testStruct.Config.OauthToken)
}

func TestScrubEmbeddedConfig(t *testing.T) {
	// Test an embedded config
	dtrIntegration := &v1.ImageIntegration{
		Name: "hi",
		IntegrationConfig: &v1.ImageIntegration_Dtr{
			Dtr: &v1.DTRConfig{
				Password: "pass",
			},
		},
	}
	ScrubSecretsFromStruct(dtrIntegration)
	assert.Empty(t, dtrIntegration.IntegrationConfig.(*v1.ImageIntegration_Dtr).Dtr.Password)
}
