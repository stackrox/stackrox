package secrets

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestScrubSecretsFromStruct(t *testing.T) {
	testStruct := &toplevel{Name: "name", Password: "password"}
	ScrubSecretsFromStructWithReplacement(testStruct, "")
	assert.Empty(t, testStruct.Password)
	assert.Equal(t, testStruct.Name, "name")
}

func TestScrubFromNestedStructPointer(t *testing.T) {
	testStruct := &toplevel{
		Name:     "name",
		Password: "password",
		ConfigPtr: &config{
			OauthToken: "oauth",
		},
		Config: config{
			OauthToken: "oauth",
		},
	}
	ScrubSecretsFromStructWithReplacement(testStruct, "")
	assert.Empty(t, testStruct.Password)
	assert.Empty(t, testStruct.ConfigPtr.OauthToken)
	assert.Empty(t, testStruct.Config.OauthToken)
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
	ScrubSecretsFromStructWithReplacement(dtrIntegration, "")
	assert.Empty(t, dtrIntegration.GetDtr().GetPassword())
}

func TestScrubSecretsWithoutPasswordSetWithReplacement(t *testing.T) {
	testStruct := &toplevel{Name: "name", Password: ""}
	ScrubSecretsFromStructWithReplacement(testStruct, ScrubReplacementStr)
	assert.Empty(t, testStruct.Password)
	assert.Equal(t, testStruct.Name, "name")
}

func TestScrubSecretsFromStructWithReplacement(t *testing.T) {
	testStruct := &toplevel{Name: "name", Password: "password"}
	ScrubSecretsFromStructWithReplacement(testStruct, ScrubReplacementStr)
	assert.Equal(t, testStruct.Password, ScrubReplacementStr)
	assert.Equal(t, testStruct.Name, "name")
}

func TestScrubFromNestedStructWithReplacement(t *testing.T) {
	testStruct := &toplevel{
		Name:     "name",
		Password: "password",
		ConfigPtr: &config{
			OauthToken: "oauth",
		},
	}
	ScrubSecretsFromStructWithReplacement(testStruct, ScrubReplacementStr)
	assert.Equal(t, testStruct.Password, ScrubReplacementStr)
	assert.Equal(t, "name", testStruct.Name)
	assert.Equal(t, ScrubReplacementStr, testStruct.ConfigPtr.OauthToken)
}

func TestScrubEmbeddedConfigWithReplacement(t *testing.T) {
	// Test an embedded config
	dtrIntegration := &storage.ImageIntegration{
		Name: "hi",
		IntegrationConfig: &storage.ImageIntegration_Dtr{
			Dtr: &storage.DTRConfig{
				Password: "pass",
			},
		},
	}
	ScrubSecretsFromStructWithReplacement(dtrIntegration, ScrubReplacementStr)
	assert.Equal(t, dtrIntegration.GetDtr().GetPassword(), ScrubReplacementStr)
}
