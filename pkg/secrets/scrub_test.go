package secrets

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
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

// validateStructTagsOnType returns error if a non-string struct field type has tag scrub:always or
// if struct field is of type interface{}
func validateStructTagsOnType(ty reflect.Type) error {
	visited := make(map[reflect.Type]struct{})
	return validateStructTagsOnTypeHelper(ty, visited)
}

func validateStructTagsOnTypeHelper(ty reflect.Type, visited map[reflect.Type]struct{}) error {
	if ty.Kind() == reflect.Ptr {
		ty = ty.Elem()
	}
	if ty.Kind() == reflect.Interface {
		return errors.Errorf("cannot walk interface field %s", ty.Name())
	}
	if ty.Kind() != reflect.Struct {
		return nil
	}
	if _, ok := visited[ty]; ok {
		return nil
	}
	visited[ty] = struct{}{}
	for i := 0; i < ty.NumField(); i++ {
		fieldType := ty.Field(i).Type
		switch fieldType.Kind() {
		case reflect.Struct, reflect.Ptr, reflect.Interface:
			if err := validateStructTagsOnTypeHelper(fieldType, visited); err != nil {
				return err
			}
		}
		fieldTag := ty.Field(i).Tag
		switch fieldTag.Get(scrubStructTag) {
		case scrubTagAlways:
			if fieldType.Kind() != reflect.String || fieldType != reflect.TypeOf("") {
				return errors.Errorf("%s:%s is not allowed on type %s",
					scrubStructTag, scrubTagAlways, fieldType)
			}
		}
	}
	return nil
}

func TestStringTypePanic(t *testing.T) {
	type stringType string
	test := struct {
		StringType stringType `scrub:"always"`
	}{"stringType"}
	assert.Error(t, validateStructTagsOnType(reflect.TypeOf(test)))
	defer func() {
		err := recover()
		assert.Contains(t, fmt.Sprint(err), "field type mismatch secrets.stringType!=string")
	}()
	ScrubSecretsFromStructWithReplacement(test, "")
}

func TestNonStringPanic(t *testing.T) {
	test := struct {
		Val int `scrub:"always"`
	}{0}
	assert.Error(t, validateStructTagsOnType(reflect.TypeOf(test)))
	defer func() {
		err := recover()
		assert.Contains(t, fmt.Sprint(err), "expected string kind, got int")
	}()
	ScrubSecretsFromStructWithReplacement(test, "")
}

func TestValidateScrubTagTypes(t *testing.T) {
	assert.Error(t, validateStructTagsOnType(reflect.TypeOf(storage.ImageIntegration{})))
	assert.NoError(t, validateStructTagsOnType(reflect.TypeOf(storage.DTRConfig{})))
	assert.NoError(t, validateStructTagsOnType(reflect.TypeOf(storage.ClairifyConfig{})))
	assert.NoError(t, validateStructTagsOnType(reflect.TypeOf(storage.DockerConfig{})))
	assert.NoError(t, validateStructTagsOnType(reflect.TypeOf(storage.QuayConfig{})))
	assert.NoError(t, validateStructTagsOnType(reflect.TypeOf(storage.ECRConfig{})))
	assert.NoError(t, validateStructTagsOnType(reflect.TypeOf(storage.TenableConfig{})))
	assert.NoError(t, validateStructTagsOnType(reflect.TypeOf(storage.GoogleConfig{})))
	assert.NoError(t, validateStructTagsOnType(reflect.TypeOf(storage.ClairConfig{})))
	assert.NoError(t, validateStructTagsOnType(reflect.TypeOf(storage.AnchoreConfig{})))
	assert.NoError(t, validateStructTagsOnType(reflect.TypeOf(storage.IBMRegistryConfig{})))

	assert.Error(t, validateStructTagsOnType(reflect.TypeOf(storage.Notifier{})))
	assert.NoError(t, validateStructTagsOnType(reflect.TypeOf(storage.Jira{})))
	assert.NoError(t, validateStructTagsOnType(reflect.TypeOf(storage.Email{})))
	assert.NoError(t, validateStructTagsOnType(reflect.TypeOf(storage.CSCC{})))
	assert.NoError(t, validateStructTagsOnType(reflect.TypeOf(storage.Splunk{})))
	assert.NoError(t, validateStructTagsOnType(reflect.TypeOf(storage.PagerDuty{})))
	assert.NoError(t, validateStructTagsOnType(reflect.TypeOf(storage.Generic{})))
	assert.NoError(t, validateStructTagsOnType(reflect.TypeOf(storage.SumoLogic{})))

	assert.NoError(t, validateStructTagsOnType(reflect.TypeOf(v1.ExchangeTokenRequest{})))
	assert.NoError(t, validateStructTagsOnType(reflect.TypeOf(storage.HTTPEndpointConfig{})))
}
