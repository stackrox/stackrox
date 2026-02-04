package postgres

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectQueryField_EnumPostTransform(t *testing.T) {

	testSchema := schema.TestStructsSchema
	var enumField *walker.Field
	for i := range testSchema.Fields {
		if testSchema.Fields[i].Name == "Enum" {
			enumField = &testSchema.Fields[i]
			break
		}
	}
	testCases := []struct {
		name          string
		searchField   string
		inputValue    int
		expectedValue string
	}{
		{
			name:          "ENUM0 conversion",
			searchField:   "Test Enum",
			inputValue:    int(storage.TestStruct_ENUM0),
			expectedValue: "ENUM0",
		},
		{
			name:          "ENUM1 conversion",
			searchField:   "Test Enum",
			inputValue:    int(storage.TestStruct_ENUM1),
			expectedValue: "ENUM1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := selectQueryField(tc.searchField, enumField, false, aggregatefunc.Unset, "")
			require.NotNil(t, result.PostTransform, "PostTransform should be set for enum fields")
			inputPtr := &tc.inputValue
			transformed := result.PostTransform(inputPtr)
			assert.Equal(t, tc.expectedValue, transformed, "PostTransform should convert *int to enum string")
		})
	}
}
