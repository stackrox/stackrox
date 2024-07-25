package jsonutil

import (
	"bytes"
	"fmt"
	"strings"
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
			"String without special characters is preserved",
			"E = mc^2",
		},
		{
			"Some special characters (<, >, &) are escaped",
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
			`\\u003c \\u003e`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			verifyJSONToProtoToJSON(t, jsonify(tc.str), []ConversionOption{})
		})

		t.Run(tc.desc+" in compact JSON", func(t *testing.T) {
			verifyJSONToProtoToJSON(t, jsonifyCompact(tc.str), []ConversionOption{OptCompact})
		})

		t.Run(tc.desc+" with OptUnEscape", func(t *testing.T) {
			verifyJSONToProtoToJSON(t, jsonify(tc.str), []ConversionOption{OptUnEscape})
		})

		t.Run(tc.desc+" in compact JSON with OptUnEscape", func(t *testing.T) {
			verifyJSONToProtoToJSON(t, jsonifyCompact(tc.str), []ConversionOption{OptUnEscape, OptCompact})
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

func verifyJSONToProtoToJSON(t *testing.T, inputJSON string, options []ConversionOption) {
	// Use ResourceByID proto because it contains a single string field.
	var proto v1.ResourceByID

	// Conversion to protobuf interprets escaped sequences if any.
	err := JSONToProto(inputJSON, &proto)
	assert.NoError(t, err)

	// Conversion to only JSON escapes characters if specified in options.
	convertedJSON, err := ProtoToJSON(&proto, options...)
	assert.NoError(t, err)
	assert.JSONEq(t, inputJSON, convertedJSON)

	if contains(options, OptCompact) {
		assert.Equal(t, inputJSON, convertedJSON)
	}
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

func TestProtoToJSONOptCompact(t *testing.T) {
	testResource := &v1.ResourceByID{Id: "test"}

	strRes, err := ProtoToJSON(testResource)
	assert.NoError(t, err)
	assert.Len(t, strings.Split(strRes, "\n"), 3)

	strRes, err = ProtoToJSON(testResource, OptCompact)
	assert.NoError(t, err)
	assert.Len(t, strings.Split(strRes, "\n"), 1)
}
