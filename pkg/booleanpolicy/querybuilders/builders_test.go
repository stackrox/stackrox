package querybuilders

import (
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestOperatorProtoMapUpToDate(t *testing.T) {
	assert.Equal(t, len(storage.BooleanOperator_value), len(operatorProtoMap))
}
