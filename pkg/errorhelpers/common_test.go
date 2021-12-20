package errorhelpers

import (
"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestCommonErrorsClass(t *testing.T) {
	assert.ErrorIs(t, ErrNoAuthzConfigured, ErrInvariantViolation)
	assert.NotErrorIs(t, ErrInvariantViolation, ErrNoAuthzConfigured)
	assert.NotErrorIs(t, ErrNoAuthzConfigured, errors.New("invariant violation"))
}

func TestExplain(t *testing.T) {
	explained := Explain(ErrNotFound, "explained")
	wrapped := errors.Wrap(explained, "and wrapped")

	assert.ErrorIs(t, explained, ErrNotFound)
	assert.ErrorIs(t, wrapped, ErrNotFound)
	assert.ErrorIs(t, wrapped, explained)
	assert.NotErrorIs(t, wrapped, Explain(ErrNotFound, "explained"))

	assert.Equal(t, ErrNotFound.Error()+": explained", explained.Error())
}

func TestOverride(t *testing.T) {
	overridden := OverrideMessage(ErrNotFound, "overridden")

	assert.ErrorIs(t, overridden, ErrNotFound)
	assert.NotErrorIs(t, ErrNotFound, overridden)

	assert.Equal(t, "overridden", overridden.Error())
}

