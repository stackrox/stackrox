package centralproxy

import (
	"testing"

	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientPolicyScopesToProtoUsesStablePathOrderAndCopiesMethods(t *testing.T) {
	scopes := tokens.HTTPRequestScopes{
		"/v1/z":      {"POST", "GET"},
		"/v1/a":      {"GET"},
		"/v1/denied": {},
	}
	result := clientPolicyScopesToProto(scopes)
	require.Len(t, result, 3)
	assert.Equal(t, "/v1/a", result[0].GetPath())
	assert.Equal(t, "/v1/denied", result[1].GetPath())
	assert.Empty(t, result[1].GetMethods())
	assert.Equal(t, "/v1/z", result[2].GetPath())
	assert.Equal(t, []string{"POST", "GET"}, result[2].GetMethods())
	scopes["/v1/z"][0] = "DELETE"
	assert.Equal(t, "POST", result[2].GetMethods()[0])
}
