package pgutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestMarshalUnmarshalRepeatedMessages(t *testing.T) {
	cases := map[string]struct {
		input []*wrapperspb.StringValue
	}{
		"nil slice":   {input: nil},
		"empty slice": {input: []*wrapperspb.StringValue{}},
		"single message": {
			input: []*wrapperspb.StringValue{wrapperspb.String("hello")},
		},
		"multiple messages": {
			input: []*wrapperspb.StringValue{
				wrapperspb.String("foo"),
				wrapperspb.String("bar"),
				wrapperspb.String("baz"),
			},
		},
		"message with empty string": {
			input: []*wrapperspb.StringValue{wrapperspb.String("")},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			data, err := MarshalRepeatedMessages(tc.input)
			require.NoError(t, err)

			result, err := UnmarshalRepeatedMessages(data, func() *wrapperspb.StringValue {
				return &wrapperspb.StringValue{}
			})
			require.NoError(t, err)

			if len(tc.input) == 0 {
				assert.Empty(t, result)
			} else {
				require.Len(t, result, len(tc.input))
				for i, msg := range result {
					assert.Equal(t, tc.input[i].GetValue(), msg.GetValue())
				}
			}
		})
	}
}

func TestUnmarshalRepeatedMessagesTruncated(t *testing.T) {
	msgs := []*wrapperspb.StringValue{wrapperspb.String("test")}
	data, err := MarshalRepeatedMessages(msgs)
	require.NoError(t, err)

	t.Run("truncated length", func(t *testing.T) {
		_, err := UnmarshalRepeatedMessages(data[:2], func() *wrapperspb.StringValue {
			return &wrapperspb.StringValue{}
		})
		assert.ErrorContains(t, err, "truncated length prefix")
	})

	t.Run("truncated data", func(t *testing.T) {
		_, err := UnmarshalRepeatedMessages(data[:5], func() *wrapperspb.StringValue {
			return &wrapperspb.StringValue{}
		})
		assert.ErrorContains(t, err, "truncated message data")
	})
}
