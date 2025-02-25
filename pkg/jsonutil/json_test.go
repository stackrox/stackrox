package jsonutil

import (
	"testing"

	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/require"
)

func TestJsonMarshaler(t *testing.T) {
	a := fixtures.GetAlert()

	expected, err := ProtoToJSON(a)
	require.NoError(t, err)
	b, err := a.MarshalJSON()
	require.NoError(t, err)

	require.JSONEq(t, expected, string(b))
}
