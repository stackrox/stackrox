package dberrors

import (
	"testing"

	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stretchr/testify/assert"
)

func TestErrNotFound(t *testing.T) {
	err := NotFound("foo", "bar")
	assert.ErrorIs(t, err, errorhelpers.ErrNotFound)
}
