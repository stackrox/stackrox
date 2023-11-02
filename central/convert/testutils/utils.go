package testutils

import (
	"bytes"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AssertProtoMessageEqual asserts the equality of two proto.Messages by marshalling them to JSON
// and comparing the JSON output.
func AssertProtoMessageEqual(t *testing.T, a, b proto.Message) {
	m := jsonpb.Marshaler{}

	jsonA := &bytes.Buffer{}
	jsonB := &bytes.Buffer{}

	require.NoError(t, m.Marshal(jsonA, a))
	require.NoError(t, m.Marshal(jsonB, b))

	// Use string for improved readability in case of test failures.
	assert.Equal(t, jsonA.String(), jsonB.String())
}
