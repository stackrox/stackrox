package jsonutil

import (
	"fmt"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestConversion(t *testing.T) {
	type conversionTestCase struct {
		desc            string
		str             string
		escaped         bool
		addUnknonwField bool
	}

	testCases := []conversionTestCase{
		{
			"String without special characters is preserved",
			"E = mc^2",
			false,
			false,
		},
		{
			"Some special characters (<, >, &) are escaped",
			"A <= B & B >= C",
			true,
			false,
		},
		{
			"String without special characters is preserved",
			"E = mc^2",
			false,
			true,
		},
		{
			"Some special characters (<, >, &) are escaped",
			"A <= B & B >= C",
			true,
			true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			// Unknown fields will be excluded in the result so we need to build a target JSON
			// that will exclude those to verify the proto conversion was good.
			inputJSON := jsonify(tc.str, tc.addUnknonwField)
			targetJSON := jsonify(tc.str, false)
			verifyJSONToProtoToJSON(t, inputJSON, targetJSON, !tc.escaped, []ConversionOption{})
		})

		t.Run(tc.desc+" in compact JSON", func(t *testing.T) {
			inputJSON := jsonifyCompact(tc.str, tc.addUnknonwField)
			targetJSON := jsonifyCompact(tc.str, false)
			verifyJSONToProtoToJSON(t, inputJSON, targetJSON, !tc.escaped, []ConversionOption{OptCompact})
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
			jsonData := jsonify(tc.str, false)
			verifyJSONToProtoToJSON(t, jsonData, jsonData, true, []ConversionOption{OptUnEscape})
		})

		t.Run(tc.desc+" in compact JSON", func(t *testing.T) {
			// Prevent JSON conversion from escaping specific charters.
			jsonData := jsonifyCompact(tc.str, false)
			verifyJSONToProtoToJSON(t, jsonData, jsonData, true, []ConversionOption{OptUnEscape, OptCompact})
		})
	}
}

// Hand-made compact JSON representation of v1.ResourceByID.
func jsonifyCompact(value string, unknownField bool) string {
	if unknownField {
		return fmt.Sprintf(`{"id":"%s","unknownfield":"junk"}`, value)
	}
	return fmt.Sprintf(`{"id":"%s"}`, value)
}

// Hand-made JSON representation of v1.ResourceByID.
func jsonify(value string, unknownField bool) string {
	if unknownField {
		return fmt.Sprintf(`{
  "id": "%s",
  "unknownfield": "junk"
}`, value)
	}
	return fmt.Sprintf(`{
  "id": "%s"
}`, value)
}

func verifyJSONToProtoToJSON(t *testing.T, inputJSON string, targetJSON string, shouldPreserve bool, options []ConversionOption) {
	// Use ResourceByID proto because it contains a single string field.
	var proto v1.ResourceByID

	// Conversion to protobuf interprets escaped sequences if any.
	err := JSONToProto(inputJSON, &proto)
	assert.NoError(t, err)

	// Conversion to only JSON escapes characters if specified in options.
	convertedJSON, err := ProtoToJSON(&proto, options...)
	assert.NoError(t, err)
	assert.Equalf(t, shouldPreserve, targetJSON == convertedJSON, "original JSON:\n%s\ntargetJSON:\n%s\nconvertedJSON:\n%s", inputJSON, targetJSON, convertedJSON)
}
