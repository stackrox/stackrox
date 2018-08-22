package search

import (
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestParseRawQuery(t *testing.T) {
	query := NewQueryBuilder().AddStrings(DeploymentName, "field1", "field12").AddStrings(Category, "field2").Query()
	expectedRequest := &v1.ParsedSearchRequest{
		Fields: map[string]*v1.ParsedSearchRequest_Values{
			DeploymentName: {
				Values: []string{"field1", "field12"},
			},
			Category: {
				Values: []string{"field2"},
			},
		},
	}

	actualRequest, err := ParseRawQuery(query)
	assert.NoError(t, err)
	assert.Equal(t, expectedRequest, actualRequest)

	// fields with raw query
	query = NewQueryBuilder().AddStrings(DeploymentName, "field1").AddStrings(Category, "field2").AddStringQuery("rawquery").Query()
	expectedRequest = &v1.ParsedSearchRequest{
		Fields: map[string]*v1.ParsedSearchRequest_Values{
			DeploymentName: {
				Values: []string{"field1"},
			},
			Category: {
				Values: []string{"field2"},
			},
		},
		StringQuery: "rawquery",
	}
	actualRequest, err = ParseRawQuery(query)
	assert.NoError(t, err)
	assert.Equal(t, expectedRequest, actualRequest)
}
