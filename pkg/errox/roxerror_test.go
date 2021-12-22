package errox

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func Test_errRox_Is(t *testing.T) {
	err0 := New(CodeNotFound, "", "not found")

	errA := New(CodeNotFound, "a", "a not found")
	errB := New(CodeNotFound, "b", "b not found")
	errAB := New(CodeNotFound, "a.b", "a.b not found")
	errC := New(CodeAlreadyExists, "a", "a not found")

	assert.ErrorIs(t, errA, err0)
	assert.ErrorIs(t, errB, err0)
	assert.ErrorIs(t, errAB, err0)
	assert.NotErrorIs(t, errC, err0)

	assert.NotErrorIs(t, errB, errA)
	assert.NotErrorIs(t, errA, errB)
	assert.NotErrorIs(t, errB, errAB)
	assert.NotErrorIs(t, errAB, errB)
	assert.NotErrorIs(t, errA, errAB)
	assert.ErrorIs(t, errAB, errA)
	assert.NotErrorIs(t, errA, errC)

	assert.ErrorIs(t, errors.WithMessage(errA, "message"), errA)
	assert.ErrorIs(t, errors.WithMessage(errAB, "message"), errA)
	assert.NotErrorIs(t, errors.WithMessage(errB, "message"), errA)

}
