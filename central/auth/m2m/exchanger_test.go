package m2m

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapStringToClaims(t *testing.T) {
	claims := map[string]interface{}{
		"sub":          "my-subject",
		"aud":          []string{"audience-1", "audience-2", "audience-3"},
		"count":        3,
		"is_org_admin": true,
		"roles": []struct {
			roles []string
		}{
			{
				[]string{"one", "two", "three"},
			},
			{
				[]string{"four", "five", "six"},
			},
		},
	}
	expectedResult := map[string][]string{
		"sub": {"my-subject"},
		"aud": {"audience-1", "audience-2", "audience-3"},
	}

	result := mapToStringClaims(claims)

	assert.Equal(t, expectedResult, result)
}
