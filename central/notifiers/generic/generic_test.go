package generic

import (
	"encoding/json"
	"io"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testAlert = fixtures.GetSerializationTestAlert()

	expectedMarshaledAlert = fixtures.GetJSONSerializedTestAlert()
)

func makeBaseGeneric() *generic {
	return &generic{
		Notifier: &storage.Notifier{
			Config: &storage.Notifier_Generic{
				Generic: &storage.Generic{},
			},
		},
	}
}

func TestConstructJSON(t *testing.T) {

	var cases = map[string]struct {
		fields []*storage.KeyValuePair
	}{
		"Base, no extra field": {},
		"Base, with extra field": {
			fields: []*storage.KeyValuePair{
				{
					Key:   "key1",
					Value: "value1",
				},
				{
					Key:   "key2",
					Value: "value2",
				},
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			notifier := makeBaseGeneric()
			var err error
			notifier.Notifier.GetGeneric().ExtraFields = c.fields
			notifier.extraFieldsJSONPrefix, err = getExtraFieldJSON(c.fields)
			require.NoError(t, err)

			reader, err := notifier.constructJSON(fixtures.GetAlert(), alertMessageKey)
			require.NoError(t, err)

			jsonTarget := make(map[string]interface{})
			assert.NoError(t, json.NewDecoder(reader).Decode(&jsonTarget))
		})
	}

	extraCases := map[string]struct {
		fields          []*storage.KeyValuePair
		expectedPayload string
	}{
		"fake alert, no extra fields": {
			expectedPayload: `{"alert":` + expectedMarshaledAlert + `}`,
		},
		"fake alert, extra fields": {
			fields: []*storage.KeyValuePair{
				{
					Key:   "key1",
					Value: "value1",
				},
				{
					Key:   "key2",
					Value: "value2",
				},
			},
			expectedPayload: `{
  "key1":"value1",
  "key2":"value2",
  "alert":` + expectedMarshaledAlert + `}`,
		},
	}

	for name, c := range extraCases {
		t.Run(name, func(t *testing.T) {
			notifier := makeBaseGeneric()

			var err error
			notifier.Notifier.GetGeneric().ExtraFields = c.fields
			notifier.extraFieldsJSONPrefix, err = getExtraFieldJSON(c.fields)
			require.NoError(t, err)

			reader, err := notifier.constructJSON(testAlert, alertMessageKey)
			require.NoError(t, err)

			jsonBytes, err := io.ReadAll(reader)
			require.NoError(t, err)

			assert.JSONEq(t, c.expectedPayload, string(jsonBytes))
		})
	}
}

func TestJSONDoesNotContainNewlines(t *testing.T) {
	var cases = map[string]struct {
		fields []*storage.KeyValuePair
	}{
		"Base, no extra field": {},
		"Base, with extra field": {
			fields: []*storage.KeyValuePair{
				{
					Key:   "key1",
					Value: "value1",
				},
				{
					Key:   "key2",
					Value: "value2",
				},
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			notifier := makeBaseGeneric()
			var err error
			notifier.Notifier.GetGeneric().ExtraFields = c.fields
			notifier.extraFieldsJSONPrefix, err = getExtraFieldJSON(c.fields)
			require.NoError(t, err)

			reader, err := notifier.constructJSON(fixtures.GetAlert(), alertMessageKey)
			require.NoError(t, err)

			jsonBytes, err := io.ReadAll(reader)
			require.NoError(t, err)
			assert.NotContains(t, string(jsonBytes), "\n")
		})
	}
}
