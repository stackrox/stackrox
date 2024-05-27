package testutils

import (
	"testing"

	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AssertProtoMessageEqual asserts the equality of two protocompat.Messages by marshalling them to JSON
// and comparing the JSON output.
func AssertProtoMessageEqual(t *testing.T, a, b protocompat.Message) {
	jsonA, errA := protocompat.MarshalToProtoJSONBytes(a)
	require.NoError(t, errA)
	jsonB, errB := protocompat.MarshalToProtoJSONBytes(b)
	require.NoError(t, errB)

	// Use string for improved readability in case of test failures.
	assert.JSONEq(t, string(jsonA), string(jsonB))
}
