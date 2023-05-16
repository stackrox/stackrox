package generic

import (
	"encoding/json"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConstructJSON(t *testing.T) {
	generic := &generic{
		Notifier: &storage.Notifier{
			Config: &storage.Notifier_Generic{
				Generic: &storage.Generic{},
			},
		},
	}

	var cases = []struct {
		fields []*storage.KeyValuePair
	}{
		{},
		{
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

	for _, c := range cases {
		var err error
		generic.Notifier.GetGeneric().ExtraFields = c.fields
		generic.extraFieldsJSONPrefix, err = getExtraFieldJSON(c.fields)
		require.NoError(t, err)

		reader, err := generic.constructJSON(fixtures.GetAlert(), alertMessageKey)
		require.NoError(t, err)

		jsonTarget := make(map[string]interface{})
		assert.NoError(t, json.NewDecoder(reader).Decode(&jsonTarget))
	}
}
