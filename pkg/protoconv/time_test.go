package protoconv

import (
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/assert"
)

func TestConvertTimeString(t *testing.T) {
	cases := []struct {
		input  string
		output *types.Timestamp
	}{
		{
			input:  "",
			output: nil,
		},
		{
			input:  "malformed",
			output: nil,
		},
		{
			input:  "2018-02-07T23:29Z",
			output: protocompat.GetProtoTimestampFromSeconds(1518046140),
		},
		{
			input:  "2019-01-20T00:00:00Z",
			output: protocompat.GetProtoTimestampFromSeconds(1547942400),
		},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			assert.Equal(t, c.output, ConvertTimeString(c.input))
		})
	}
}
