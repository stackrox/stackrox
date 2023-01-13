package jsonutil

import (
	"bytes"
	"fmt"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestConversion(t *testing.T) {
	type conversionTestCase struct {
		desc    string
		str     string
		escaped bool
	}

	testCases := []conversionTestCase{
		{
			"String without special characters is preserved",
			"E = mc^2",
			false,
		},
		{
			"Some special characters (<, >, &) are escaped",
			"A <= B & B >= C",
			true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			verifyJSONToProtoToJSON(t, jsonify(tc.str), !tc.escaped, []ConversionOption{})
		})

		t.Run(tc.desc+" in compact JSON", func(t *testing.T) {
			verifyJSONToProtoToJSON(t, jsonifyCompact(tc.str), !tc.escaped, []ConversionOption{OptCompact})
		})
	}
}

func TestConversionWithUnEscape(t *testing.T) {
	type conversionTestCase struct {
		desc string
		str  string
	}

	testCases := []conversionTestCase{
		{
			"Single <, >, & are preserved",
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

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			verifyJSONToProtoToJSON(t, jsonify(tc.str), true, []ConversionOption{OptUnEscape})
		})

		t.Run(tc.desc+" in compact JSON", func(t *testing.T) {
			// Prevent JSON conversion from escaping specific charters.
			verifyJSONToProtoToJSON(t, jsonifyCompact(tc.str), true, []ConversionOption{OptUnEscape, OptCompact})
		})
	}
}

// Hand-made compact JSON representation of v1.ResourceByID.
func jsonifyCompact(value string) string {
	return fmt.Sprintf(`{"id":"%s"}`, value)
}

// Hand-made JSON representation of v1.ResourceByID.
func jsonify(value string) string {
	return fmt.Sprintf(`{
  "id": "%s"
}`, value)
}

func verifyJSONToProtoToJSON(t *testing.T, inputJSON string, shouldPreserve bool, options []ConversionOption) {
	// Use ResourceByID proto because it contains a single string field.
	var proto v1.ResourceByID

	// Conversion to protobuf interprets escaped sequences if any.
	err := JSONToProto(inputJSON, &proto)
	assert.NoError(t, err)

	// Conversion to only JSON escapes characters if specified in options.
	convertedJSON, err := ProtoToJSON(&proto, options...)
	assert.NoError(t, err)
	assert.Equalf(t, shouldPreserve, inputJSON == convertedJSON, "original JSON:\n%s\nconvertedJSON:\n%s", inputJSON, convertedJSON)
}

func TestNoErrorOnUnknownAttribute(t *testing.T) {
	const json = `{ "id": "6500", "unknownField": "junk" }`
	var proto v1.ResourceByID

	err := JSONToProto(json, &proto)
	assert.NoError(t, err)
	assert.Equal(t, "6500", proto.GetId())

	jsonBytes := []byte(json)

	err = JSONBytesToProto(jsonBytes, &proto)
	assert.NoError(t, err)
	assert.Equal(t, "6500", proto.GetId())

	err = JSONReaderToProto(bytes.NewReader(jsonBytes), &proto)
	assert.NoError(t, err)
	assert.Equal(t, "6500", proto.GetId())
}
