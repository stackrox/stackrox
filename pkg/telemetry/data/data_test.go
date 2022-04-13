package data

import (
	"encoding/json"
	"reflect"
	"regexp"
	"testing"

	"github.com/stackrox/stackrox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	fieldNameRegex = regexp.MustCompile(`^[a-z]([a-zA-Z0-9]*)$`)
)

func validateFieldName(t *testing.T, fieldName string) {
	assert.Truef(t, fieldNameRegex.MatchString(fieldName), "Field name %q does not match expected field name pattern", fieldName)
}

func validateJSONFieldNames(t *testing.T, val reflect.Value) {
	switch val.Kind() {
	case reflect.Slice:
		for i := 0; i < val.Len(); i++ {
			validateJSONFieldNames(t, val.Index(i))
		}
	case reflect.Map:
		m, ok := val.Interface().(map[string]interface{})
		assert.True(t, ok)
		for mKey, mVal := range m {
			validateFieldName(t, mKey)
			validateJSONFieldNames(t, reflect.ValueOf(mVal))
		}
	}
}

func TestJSONSerialization_FieldsHaveValidNames(t *testing.T) {
	var d TelemetryData
	require.NoError(t, testutils.FullInit(&d, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))

	jsonBytes, err := json.Marshal(&d)
	require.NoError(t, err)

	var generic map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonBytes, &generic))

	validateJSONFieldNames(t, reflect.ValueOf(generic))
}

func TestJSONSerialization_MarshalUnmarshalIsIdentity(t *testing.T) {
	var d TelemetryData
	require.NoError(t, testutils.FullInit(&d, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))

	jsonBytes, err := json.Marshal(&d)
	require.NoError(t, err)

	var d2 TelemetryData
	require.NoError(t, json.Unmarshal(jsonBytes, &d2))

	assert.Equal(t, d, d2)
}
