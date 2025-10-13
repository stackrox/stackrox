package protoutils

import (
	"reflect"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestMessageType(t *testing.T) {
	tests := []struct {
		given    string
		expected reflect.Type
	}{
		{given: "", expected: nil},
		{given: "unknown.type", expected: nil},
		{given: "storage.Cluster", expected: reflect.TypeOf(&storage.Cluster{})},
		{given: "google.protobuf.Any", expected: reflect.TypeOf(&anypb.Any{})},
	}
	for _, tt := range tests {
		t.Run(tt.given, func(t *testing.T) {
			actual := MessageType(tt.given)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
