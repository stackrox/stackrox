package secrets

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

type config struct {
	OauthToken string `scrub:"always"`
}

type toplevel struct {
	Name     string
	Password string `scrub:"always"`
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
}

func TestScrubEmbeddedConfig(t *testing.T) {
	// Test an embedded config
	dtrIntegration := &storage.ImageIntegration{
		Name: "hi",
		IntegrationConfig: &storage.ImageIntegration_Dtr{
			Dtr: &storage.DTRConfig{
				Password: "pass",
			},
		},
	}
	ScrubSecretsFromStruct(dtrIntegration)
	assert.Empty(t, dtrIntegration.GetDtr().GetPassword())
}

func TestScrubSecretsWithoutPasswordSetWithEncryption(t *testing.T) {
	testStruct := &toplevel{Name: "name", Password: ""}
	ScrubSecretsFromStructWithReplacement(testStruct, ReplacementStr)
	assert.Empty(t, testStruct.Password)
	assert.Equal(t, testStruct.Name, "name")
}

func TestScrubSecretsFromStructWithEncryption(t *testing.T) {
	testStruct := &toplevel{Name: "name", Password: "password"}
	ScrubSecretsFromStructWithReplacement(testStruct, ReplacementStr)
	assert.Equal(t, testStruct.Password, ReplacementStr)
	assert.Equal(t, testStruct.Name, "name")
}

func TestScrubFromNestedStructWithEncryption(t *testing.T) {
	testStruct := &toplevel{
		Name:     "name",
		Password: "password",
		Config: &config{
			OauthToken: "oauth",
		},
	}
	ScrubSecretsFromStructWithReplacement(testStruct, ReplacementStr)
	assert.Equal(t, testStruct.Password, ReplacementStr)
	assert.Equal(t, "name", testStruct.Name)
	assert.Equal(t, ReplacementStr, testStruct.Config.OauthToken)
}

func TestScrubEmbeddedConfigWithEncryption(t *testing.T) {
	// Test an embedded config
	dtrIntegration := &storage.ImageIntegration{
		Name: "hi",
		IntegrationConfig: &storage.ImageIntegration_Dtr{
			Dtr: &storage.DTRConfig{
				Password: "pass",
			},
		},
	}
	ScrubSecretsFromStructWithReplacement(dtrIntegration, ReplacementStr)
	assert.Equal(t, dtrIntegration.GetDtr().GetPassword(), ReplacementStr)
}
