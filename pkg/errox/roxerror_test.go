package errox

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func Test_errRox_Is(t *testing.T) {
	errA := New(CodeNotFound, "a", "a not found")
	errB := New(CodeNotFound, "b", "b not found")
	errAB := New(CodeNotFound, "a.b", "a.b not found")
	errC := New(CodeAlreadyExists, "a", "a not found")

	assert.False(t, errors.Is(errB, errA))
	assert.False(t, errors.Is(errA, errB))
	assert.False(t, errors.Is(errB, errAB))
	assert.False(t, errors.Is(errAB, errB))
	assert.False(t, errors.Is(errA, errAB))
	assert.True(t, errors.Is(errAB, errA))
	assert.False(t, errors.Is(errA, errC))

	assert.True(t, errors.Is(errors.WithMessage(errA, "message"), errA))
	assert.True(t, errors.Is(errors.WithMessage(errAB, "message"), errA))
	assert.False(t, errors.Is(errors.WithMessage(errB, "message"), errA))

}
