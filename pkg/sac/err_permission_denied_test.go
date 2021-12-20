package sac

import (
	"testing"

	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stretchr/testify/assert"
)

func TestErrResourceAccessDeniedClass(t *testing.T) {
	assert.ErrorIs(t, ErrResourceAccessDenied, errorhelpers.ErrNotAuthorized)
	assert.NotErrorIs(t, errorhelpers.ErrNotAuthorized, ErrResourceAccessDenied)
}
