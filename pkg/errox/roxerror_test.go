package errox

import (
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func Test_errRox_Is(t *testing.T) {
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

func TestWrap(t *testing.T) {
	{
		err := NotFound
		err = Wrap(err, NotAuthorized)

		assert.NotErrorIs(t, err, InvalidArgs)
		assert.ErrorIs(t, err, NotFound)
		assert.ErrorIs(t, err, NotAuthorized)
	}

	{
		mine := Wrap(New(NotFound, "cannot load"), NotAuthorized)
		assert.ErrorIs(t, mine, NotAuthorized)
	}

	{
		mine := Wrap(errors.New("cannot load"), NotAuthorized)
		assert.ErrorIs(t, mine, NotAuthorized)
	}
	assert.ErrorIs(t, Wrap(os.ErrNotExist, NotFound), os.ErrNotExist)
}

func TestError(t *testing.T) {
	{
		err := NotFound
		assert.Equal(t, "not found", err.Error())
		err = Wrap(err, NotAuthorized)
		assert.Equal(t, "not authorized: not found", err.Error())
	}

	{
		mine := New(NotFound, "cannot load")
		assert.Equal(t, "cannot load", mine.Error())
	}

	{
		mine := Wrap(New(NotFound, "cannot load"), NotAuthorized)
		assert.Equal(t, "not authorized: cannot load", mine.Error())
	}

	{
		mine := Wrap(errors.New("cannot load"), NotAuthorized)
		assert.Equal(t, "not authorized: cannot load", mine.Error())
	}

	{
		err := errors.New("no such file config.yaml")
		err = Wrap(err, NotFound)
		err = errors.WithMessage(err, "Opening settings")
		assert.Equal(t, "Opening settings: not found: no such file config.yaml", err.Error())
	}
}

func TestErrorAs(t *testing.T) {
	err := errors.Wrap(NotFound, "wrapped")
	var re RoxError
	assert.ErrorAs(t, err, &re)
	assert.Equal(t, "not found", re.Error())
}
