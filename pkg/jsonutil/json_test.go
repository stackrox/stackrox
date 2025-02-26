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

var input = []protoWithJson{
	fixtures.GetAlert(),
	fixtures.GetImage(),
	fixtures.GetCluster("cluster"),
	fixtures.GetImageAlert(),
	fixtures.GetDeployment(),
}

func TestJsonMarshaler(t *testing.T) {
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

//func BenchmarkProtoToJSON(b *testing.B) {
//	for _, msg := range input {
//		b.Run(string(msg.ProtoReflect().Type().Descriptor().FullName()), func(b *testing.B) {
//			for i := 0; i < b.N; i++ {
//				expected, err := ProtoToJSON(msg)
//				require.NoError(b, err)
//				require.NotEmpty(b, expected)
//			}
//		})
//	}
//}

func BenchmarkProtoToJSON(b *testing.B) {
	for _, msg := range input {
		b.Run(string(msg.ProtoReflect().Type().Descriptor().FullName()), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				j, err := msg.MarshalJSON()
				require.NoError(b, err)
				require.NotEmpty(b, j)
			}
		})
	}
}
