package common

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestConversion(t *testing.T) {
	type conversionTestCase struct {
		desc string
		str  string
	}

	testCases := []conversionTestCase{
		{
			"single <, >, & are preserved",
			"A <= B & B >= C",
		},
		{
			"Symbols are preserved at the start and end of the string",
			"<&>",
		},
		{
			"Repetitions are preserved",
			">>> &&& <<<",
		},
		{
			"Non-escaped but similar sequences are untouched",
			`\\u003c \\\u003e`,
		},
	}

	// Hand-made JSON representation of v1.ResourceByID.
	jsonify := func(value string) string {
		return "{\n  \"id\": \"" + value + "\"\n}"
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			// Reference JSON representation.
			expectedJSON := jsonify(tc.str)

			// Use ResourceByID proto because it contains a single string field.
			var proto v1.ResourceByID

			// Conversion to protobuf interprets escaped sequences.
			err := JSONToProto(expectedJSON, &proto)
			assert.NoError(t, err)

			// Conversion to JSON escapes specific charters.
			json, err := ProtoToJSON(&proto)
			assert.NoError(t, err)

			unescapedJSON := UnEscape(json)
			assert.Equal(t, expectedJSON, unescapedJSON)
		})
	}
}
