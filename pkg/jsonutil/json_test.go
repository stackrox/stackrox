package jsonutil

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/require"
)

type protoWithJson interface {
	protocompat.Message
	json.Marshaler
}

func TestJsonMarshaler(t *testing.T) {

	input := []protoWithJson{
		fixtures.GetAlert(),
		fixtures.GetImage(),
		fixtures.GetCluster("cluster"),
		fixtures.GetImageAlert(),
		fixtures.GetDeployment(),
	}

	for _, msg := range input {
		t.Run(string(msg.ProtoReflect().Type().Descriptor().FullName()), func(t *testing.T) {
			expected, err := ProtoToJSON(msg)
			require.NoError(t, err)

			b, err := msg.MarshalJSON()
			require.NoError(t, err)

			var dest bytes.Buffer
			err = json.Indent(&dest, b, "", "  ")
			require.NoError(t, err, string(b))
			t.Log(dest.String())
			require.JSONEq(t, expected, string(b))
		})
	}
}
