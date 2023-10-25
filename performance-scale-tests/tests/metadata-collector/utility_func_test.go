package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetFlagName(t *testing.T) {
	testCases := map[string]struct {
		inputField     *MetadataField
		expectedOutput string
	}{
		"nil safety": {
			inputField:     nil,
			expectedOutput: "",
		},
		"basic lower case string": {
			inputField: &MetadataField{
				Name: "abcd",
			},
			expectedOutput: "abcd",
		},
		"ignore characters that are neither letter nor number": {
			inputField: &MetadataField{
				Name: "a*b=c 🙂",
			},
			expectedOutput: "abc",
		},
		"camel case is split into dash-separated lower-case words": {
			inputField: &MetadataField{
				Name: "camelCaseIsNotFlag",
			},
			expectedOutput: "camel-case-is-not-flag",
		},
		"multiple consecutive upper case are part of the same word": {
			inputField: &MetadataField{
				Name: "camelCaseAPI",
			},
			expectedOutput: "camel-case-api",
		},
		"digits are part of words": {
			inputField: &MetadataField{
				Name: "C0wB0yz",
			},
			expectedOutput: "c0w-b0yz",
		},
	}

	for name, data := range testCases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, data.expectedOutput, data.inputField.getFlagName())
		})
	}
}
