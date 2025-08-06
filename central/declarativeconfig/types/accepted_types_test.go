package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSupportedTypes(t *testing.T) {
	assert.Len(t, GetSupportedProtobufTypesInProcessingOrder(), 7)
}
