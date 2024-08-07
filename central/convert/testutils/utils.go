package testutils

import (
	"testing"

	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

// AssertProtoMessageEqual asserts the equality of two protocompat.Messages by marshalling them to JSON
// and comparing the JSON output.
func AssertProtoMessageEqual(t *testing.T, a, b protocompat.Message) {
	m := protojson.MarshalOptions{}

	jsonA, err := m.Marshal(a)
	require.NoError(t, err)
	jsonB, err := m.Marshal(b)
	require.NoError(t, err)

	// Use string for improved readability in case of test failures.
	assert.JSONEq(t, string(jsonA), string(jsonB))
}
