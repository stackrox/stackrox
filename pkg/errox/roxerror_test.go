package errox

import (
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestRoxErrorIs(t *testing.T) {
	errNotFound := makeSentinel("base not found")
	errNotFound1 := makeSentinel("base not found")

	assert.NotErrorIs(t, errNotFound, errNotFound1)

	errA := New(errNotFound, "a not found")
	errA1 := New(errNotFound, "a not found")
	errB := New(errNotFound, "b not found")
	errAA := New(errA, "aa not found")

	assert.ErrorIs(t, errA, errNotFound)
	assert.ErrorIs(t, errB, errNotFound)
	assert.ErrorIs(t, errAA, errNotFound)
	assert.ErrorIs(t, errNotFound, errNotFound)
	assert.NotErrorIs(t, AlreadyExists, errNotFound)
	assert.NotErrorIs(t, errA, nil)
	assert.NotErrorIs(t, errA, errA1)

	assert.NotErrorIs(t, errB, errA)
	assert.NotErrorIs(t, errA, errB)
	assert.NotErrorIs(t, errB, errAA)
	assert.NotErrorIs(t, errAA, errB)
	assert.NotErrorIs(t, errA, errAA)
	assert.ErrorIs(t, errAA, errA)
	assert.NotErrorIs(t, errA, AlreadyExists)

	assert.ErrorIs(t, errors.WithMessage(errA, "message"), errA)
	assert.ErrorIs(t, errors.WithMessage(errAA, "message"), errA)
	assert.NotErrorIs(t, errors.WithMessage(errB, "message"), errA)

	assert.NotErrorIs(t, errNotFound, errors.New("some error"))
	assert.NotErrorIs(t, errors.New("some error"), errNotFound)
}

func TestErrorMessage(t *testing.T) {
	{
		err := NotFound
		assert.Equal(t, "not found", err.Error())
	}

	{
		mine := New(NotFound, "cannot load")
		assert.Equal(t, "cannot load", mine.Error())
	}

	{
		err := Newf(InvalidArgs, "custom %s", "message")
		assert.Equal(t, "custom message", err.Error())
		assert.ErrorIs(t, err, InvalidArgs)
	}

	{
		err := New(os.ErrClosed, "not open")
		assert.Equal(t, "not open", err.Error())
		assert.ErrorIs(t, err, os.ErrClosed)
	}
}
