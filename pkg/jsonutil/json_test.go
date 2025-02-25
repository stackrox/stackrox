package jsonutil

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestJsonMarshaler(t *testing.T) {
	a := fixtures.GetAlert()

	marshaller := &protojson.MarshalOptions{
		EmitUnpopulated: true,
	}
	expected, err := marshaller.Marshal(a)
	require.NoError(t, err)

	b, err := a.MarshalJSON()
	require.NoError(t, err)

	var dest bytes.Buffer
	err = json.Indent(&dest, b, "", "  ")
	require.NoError(t, err)
	t.Log(dest.String())
	require.JSONEq(t, string(expected), string(b))
}
