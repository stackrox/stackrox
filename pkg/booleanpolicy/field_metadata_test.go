package booleanpolicy

import (
	"testing"

	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stretchr/testify/assert"
)

func TestAllFieldsHaveMetadata(t *testing.T) {
	assert.Equal(t, fieldnames.Count(), len(fieldsToQB))
}
