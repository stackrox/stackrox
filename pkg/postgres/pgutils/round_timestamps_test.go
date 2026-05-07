package pgutils

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestRoundTimestampsToMicroseconds(t *testing.T) {
	tests := map[string]struct {
		input    *timestamppb.Timestamp
		expected *timestamppb.Timestamp
	}{
		"nil timestamp": {
			input:    nil,
			expected: nil,
		},
		"already aligned to microsecond": {
			input:    &timestamppb.Timestamp{Seconds: 1, Nanos: 123456000},
			expected: &timestamppb.Timestamp{Seconds: 1, Nanos: 123456000},
		},
		"rounds up when >= 500 nanos": {
			input:    &timestamppb.Timestamp{Seconds: 1, Nanos: 62244753},
			expected: &timestamppb.Timestamp{Seconds: 1, Nanos: 62245000},
		},
		"rounds down when < 500 nanos": {
			input:    &timestamppb.Timestamp{Seconds: 1, Nanos: 123456499},
			expected: &timestamppb.Timestamp{Seconds: 1, Nanos: 123456000},
		},
		"rounds down at exactly 499": {
			input:    &timestamppb.Timestamp{Seconds: 1, Nanos: 123456499},
			expected: &timestamppb.Timestamp{Seconds: 1, Nanos: 123456000},
		},
		"rounds up at exactly 500": {
			input:    &timestamppb.Timestamp{Seconds: 1, Nanos: 123456500},
			expected: &timestamppb.Timestamp{Seconds: 1, Nanos: 123457000},
		},
		"handles overflow into seconds": {
			input:    &timestamppb.Timestamp{Seconds: 1, Nanos: 999999999},
			expected: &timestamppb.Timestamp{Seconds: 2, Nanos: 0},
		},
		"zero nanos": {
			input:    &timestamppb.Timestamp{Seconds: 1, Nanos: 0},
			expected: &timestamppb.Timestamp{Seconds: 1, Nanos: 0},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.input != nil {
				RoundTimestampsToMicroseconds(tc.input)
				assert.Equal(t, tc.expected.GetNanos(), tc.input.GetNanos())
				assert.Equal(t, tc.expected.GetSeconds(), tc.input.GetSeconds())
			}
		})
	}
}

func TestRoundTimestampsInNestedStructures(t *testing.T) {
	// Test with a real StackRox storage message that has multiple timestamp fields
	alert := &storage.Alert{
		Id: "test-alert",
		Time: &timestamppb.Timestamp{
			Seconds: 1000,
			Nanos:   123456789, // Has sub-microsecond precision
		},
		FirstOccurred: &timestamppb.Timestamp{
			Seconds: 2000,
			Nanos:   987654321,
		},
		ResolvedAt: &timestamppb.Timestamp{
			Seconds: 3000,
			Nanos:   111222333,
		},
	}

	RoundTimestampsToMicroseconds(alert)

	// All timestamps should be rounded to microsecond precision
	assert.Equal(t, int32(123457000), alert.GetTime().GetNanos())          // 789 rounds up
	assert.Equal(t, int32(987654000), alert.GetFirstOccurred().GetNanos()) // 321 rounds down
	assert.Equal(t, int32(111222000), alert.GetResolvedAt().GetNanos())    // 333 rounds down

	// Seconds should remain unchanged
	assert.Equal(t, int64(1000), alert.GetTime().GetSeconds())
	assert.Equal(t, int64(2000), alert.GetFirstOccurred().GetSeconds())
	assert.Equal(t, int64(3000), alert.GetResolvedAt().GetSeconds())
}
