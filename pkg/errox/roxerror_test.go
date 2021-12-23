package errox

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func Test_errRox_Is(t *testing.T) {
	errA := NewCustom(NotFound, "a not found")
	errA1 := NewCustom(NotFound, "a not found")
	errB := NewCustom(NotFound, "b not found")
	errAA := NewCustom(errA, "aa not found")

	assert.ErrorIs(t, errA, NotFound)
	assert.ErrorIs(t, errB, NotFound)
	assert.ErrorIs(t, errAA, NotFound)
	assert.ErrorIs(t, NotFound, NotFound)
	assert.NotErrorIs(t, AlreadyExists, NotFound)
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

}

func TestWrap(t *testing.T) {
	{
		err := Wrap(NotFound, NotAuthorized)
		assert.NotErrorIs(t, err, InvalidArgs)
		assert.ErrorIs(t, err, NotFound)
		assert.ErrorIs(t, err, NotAuthorized)

		assert.Equal(t, "not authorized: not found", err.Error())
	}

	{
		mine := Wrap(errors.New("cannot load"), NotAuthorized)
		assert.ErrorIs(t, mine, NotAuthorized)
		assert.Equal(t, "not authorized: cannot load", mine.Error())
	}
}
