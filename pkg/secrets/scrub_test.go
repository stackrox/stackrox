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

func TestScrubMapFromStruct(t *testing.T) {
	testStruct := &toplevel{
		Map:      map[string]string{"client_secret": "scrub", "keep": "keep_top"},
		ScrubMap: map[string]string{"client_secret": "scrub", "keep": "keep_top"},
		ConfigPtr: &config{
			Map:      map[string]string{"client_secret": "scrub", "keep": "keep_ptr"},
			ScrubMap: map[string]string{"client_secret": "scrub", "keep": "keep_ptr"},
		},
		Config: config{
			Map:      map[string]string{"client_secret": "scrub", "keep": "keep"},
			ScrubMap: map[string]string{"client_secret": "scrub", "keep": "keep"},
		},
	}
	ScrubSecretsFromStructWithReplacement(testStruct, "***")

	assert.Equal(t, map[string]string{"client_secret": "scrub", "keep": "keep_top"}, testStruct.Map)
	assert.Equal(t, map[string]string{"client_secret": "***", "keep": "keep_top"}, testStruct.ScrubMap)
	assert.Equal(t, map[string]string{"client_secret": "scrub", "keep": "keep_ptr"}, testStruct.ConfigPtr.Map)
	assert.Equal(t, map[string]string{"client_secret": "***", "keep": "keep_ptr"}, testStruct.ConfigPtr.ScrubMap)
	assert.Equal(t, map[string]string{"client_secret": "scrub", "keep": "keep"}, testStruct.Config.Map)
	assert.Equal(t, map[string]string{"client_secret": "***", "keep": "keep"}, testStruct.Config.ScrubMap)
}

func TestScrubMapFromStructWithoutChange(t *testing.T) {
	testStruct := &toplevel{
		Map:      map[string]string{"keep": "keep_top"},
		ScrubMap: map[string]string{"keep": "keep_top"},
		ConfigPtr: &config{
			Map:      map[string]string{"keep": "keep_ptr"},
			ScrubMap: map[string]string{"keep": "keep_ptr"},
		},
		Config: config{
			Map:      map[string]string{"keep": "keep"},
			ScrubMap: map[string]string{"keep": "keep"},
		},
	}
	ScrubSecretsFromStructWithReplacement(testStruct, "***")

	assert.Equal(t, map[string]string{"keep": "keep_top"}, testStruct.Map)
	assert.Equal(t, map[string]string{"keep": "keep_top"}, testStruct.ScrubMap)
	assert.Equal(t, map[string]string{"keep": "keep_ptr"}, testStruct.ConfigPtr.Map)
	assert.Equal(t, map[string]string{"keep": "keep_ptr"}, testStruct.ConfigPtr.ScrubMap)
	assert.Equal(t, map[string]string{"keep": "keep"}, testStruct.Config.Map)
	assert.Equal(t, map[string]string{"keep": "keep"}, testStruct.Config.ScrubMap)
}

func TestScrubMapFromStructEmpty(t *testing.T) {
	testStruct := &toplevel{
		Map:      map[string]string{},
		ScrubMap: map[string]string{},
		ConfigPtr: &config{
			Map:      map[string]string{},
			ScrubMap: map[string]string{},
		},
		Config: config{
			Map:      map[string]string{},
			ScrubMap: map[string]string{},
		},
	}
	ScrubSecretsFromStructWithReplacement(testStruct, "***")

	assert.Empty(t, testStruct.Map)
	assert.Empty(t, testStruct.ScrubMap)
	assert.Empty(t, testStruct.ConfigPtr.Map)
	assert.Empty(t, testStruct.ConfigPtr.ScrubMap)
	assert.Empty(t, testStruct.Config.Map)
	assert.Empty(t, testStruct.Config.ScrubMap)
}

func TestScrubNilMapFromStructWithoutChange(t *testing.T) {
	testStruct := &toplevel{
		Map:      nil,
		ScrubMap: nil,
		ConfigPtr: &config{
			Map:      nil,
			ScrubMap: nil,
		},
		Config: config{
			Map:      nil,
			ScrubMap: nil,
		},
	}
	ScrubSecretsFromStructWithReplacement(testStruct, "***")

	assert.Nil(t, testStruct.Map)
	assert.Nil(t, testStruct.ScrubMap)
	assert.Nil(t, testStruct.ConfigPtr.Map)
	assert.Nil(t, testStruct.ConfigPtr.ScrubMap)
	assert.Nil(t, testStruct.Config.Map)
	assert.Nil(t, testStruct.Config.ScrubMap)
}

func TestScrubMapFromStructNotSupportedMap(t *testing.T) {
	type notSupportedKeyType struct {
		ScrubMap map[int]string `scrub:"map-values"`
	}

	assert.Panics(t, func() { ScrubSecretsFromStructWithReplacement(notSupportedKeyType{}, "") })
	assert.Panics(t, func() {
		ScrubSecretsFromStructWithReplacement(notSupportedKeyType{ScrubMap: map[int]string{1: "one"}}, "")
	})

	type nestedNotSupportedKeyType struct {
		Nested    notSupportedKeyType
		NestedPtr *notSupportedKeyType
	}

	assert.Panics(t, func() { ScrubSecretsFromStructWithReplacement(nestedNotSupportedKeyType{}, "") })
	assert.Panics(t, func() {
		ScrubSecretsFromStructWithReplacement(nestedNotSupportedKeyType{Nested: notSupportedKeyType{ScrubMap: map[int]string{1: "one"}}}, "")
	})
	assert.Panics(t, func() {
		ScrubSecretsFromStructWithReplacement(nestedNotSupportedKeyType{NestedPtr: &notSupportedKeyType{ScrubMap: map[int]string{1: "one"}}}, "")
	})

	type notSupportedValueType struct {
		ScrubMap map[string]int `scrub:"map-values"`
	}

	assert.Panics(t, func() { ScrubSecretsFromStructWithReplacement(notSupportedValueType{}, "") })
	assert.Panics(t, func() {
		ScrubSecretsFromStructWithReplacement(notSupportedValueType{ScrubMap: map[string]int{"one": 1}}, "")
	})

	type nestedNotSupportedValueType struct {
		Nested    notSupportedValueType
		NestedPtr *notSupportedValueType
	}

	assert.Panics(t, func() { ScrubSecretsFromStructWithReplacement(nestedNotSupportedValueType{}, "") })
	assert.Panics(t, func() {
		ScrubSecretsFromStructWithReplacement(nestedNotSupportedValueType{Nested: notSupportedValueType{ScrubMap: map[string]int{"one": 1}}}, "")
	})
	assert.Panics(t, func() {
		ScrubSecretsFromStructWithReplacement(nestedNotSupportedValueType{NestedPtr: &notSupportedValueType{ScrubMap: map[string]int{"one": 1}}}, "")
	})
}

func TestScrubMapFromStructNotSupportedMapWithoutTag(t *testing.T) {
	type noTagUnsupportedType struct {
		MapKey   map[int]string
		MapValue map[string]int
	}

	testStruct := noTagUnsupportedType{
		MapKey:   map[int]string{1: "one"},
		MapValue: map[string]int{"two": 2},
	}
	ScrubSecretsFromStructWithReplacement(testStruct, "")

	assert.Equal(t, map[int]string{1: "one"}, testStruct.MapKey)
	assert.Equal(t, map[string]int{"two": 2}, testStruct.MapValue)

	type noTagUnsupportedTypeNested struct {
		Nested    noTagUnsupportedType
		NestedPtr *noTagUnsupportedType
	}

	testNestedStruct := &noTagUnsupportedTypeNested{
		Nested: noTagUnsupportedType{
			MapKey:   map[int]string{1: "one"},
			MapValue: map[string]int{"two": 2},
		},
		NestedPtr: &noTagUnsupportedType{
			MapKey:   map[int]string{1: "one"},
			MapValue: map[string]int{"two": 2},
		},
	}
	ScrubSecretsFromStructWithReplacement(testNestedStruct, "")

	assert.Equal(t, map[int]string{1: "one"}, testNestedStruct.Nested.MapKey)
	assert.Equal(t, map[string]int{"two": 2}, testNestedStruct.Nested.MapValue)

	assert.Equal(t, map[int]string{1: "one"}, testNestedStruct.NestedPtr.MapKey)
	assert.Equal(t, map[string]int{"two": 2}, testNestedStruct.NestedPtr.MapValue)
}

func TestScrubEmbeddedConfig(t *testing.T) {
	// Test an embedded config
	ecrIntegration := &storage.ImageIntegration{
		Name: "hi",
		IntegrationConfig: &storage.ImageIntegration_Ecr{
			Ecr: &storage.ECRConfig{
				SecretAccessKey: "key",
			},
		},
	}
	ScrubSecretsFromStructWithReplacement(ecrIntegration, "")
	assert.Empty(t, ecrIntegration.GetEcr().GetSecretAccessKey())
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
	ecrIntegration := &storage.ImageIntegration{
		Name: "hi",
		IntegrationConfig: &storage.ImageIntegration_Ecr{
			Ecr: &storage.ECRConfig{
				SecretAccessKey: "key",
			},
		},
	}
	ScrubSecretsFromStructWithReplacement(ecrIntegration, ScrubReplacementStr)
	assert.Equal(t, ecrIntegration.GetEcr().GetSecretAccessKey(), ScrubReplacementStr)
}

func TestScrubFromStructWithOneOf(t *testing.T) {
	impl := &oneOfImplementation{
		Secret: "iamasecret",
	}
	wrapper := OneOfWrapper{
		SecretInterface: impl,
	}
	ScrubSecretsFromStructWithReplacement(wrapper, ScrubReplacementStr)
	assert.Equal(t, impl.Secret, ScrubReplacementStr)
}

type OneOfInterface interface {
	isOneOf()
}

type OneOfWrapper struct {
	SecretInterface OneOfInterface
}

type oneOfImplementation struct {
	Secret string `scrub:"always"`
}

func (o *oneOfImplementation) isOneOf() {}

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
	if ty.Kind() == reflect.Interface || ty.Kind() != reflect.Struct {
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
	assert.NoError(t, validateStructTagsOnType(reflect.TypeOf(storage.ImageIntegration{})))
	assert.NoError(t, validateStructTagsOnType(reflect.TypeOf(storage.ClairifyConfig{})))
	assert.NoError(t, validateStructTagsOnType(reflect.TypeOf(storage.DockerConfig{})))
	assert.NoError(t, validateStructTagsOnType(reflect.TypeOf(storage.QuayConfig{})))
	assert.NoError(t, validateStructTagsOnType(reflect.TypeOf(storage.ECRConfig{})))
	assert.NoError(t, validateStructTagsOnType(reflect.TypeOf(storage.GoogleConfig{})))
	assert.NoError(t, validateStructTagsOnType(reflect.TypeOf(storage.ClairV4Config{})))
	assert.NoError(t, validateStructTagsOnType(reflect.TypeOf(storage.IBMRegistryConfig{})))

	assert.NoError(t, validateStructTagsOnType(reflect.TypeOf(storage.Notifier{})))
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
