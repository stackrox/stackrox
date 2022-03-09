package errox

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestRoxErrorIs(t *testing.T) {
	errNotFound := makeSentinel("base not found")
	assert.NotErrorIs(t, AlreadyExists, errNotFound)
	assert.ErrorIs(t, errNotFound, errNotFound)
	assert.NotErrorIs(t, errNotFound, errors.New("some error"))
	assert.NotErrorIs(t, errors.New("some error"), errNotFound)

	errNotFound1 := makeSentinel("base not found")
	assert.NotErrorIs(t, errNotFound, errNotFound1)

	fileNotFound := errNotFound.New("file not found")
	cpuNotFound := errNotFound.New("CPU not found")
	googleNotFound := errNotFound.New("Google not found")
	movieNotFound := fileNotFound.New("movie not found")

	assert.ErrorIs(t, fileNotFound, errNotFound)
	assert.ErrorIs(t, googleNotFound, errNotFound)
	assert.ErrorIs(t, movieNotFound, fileNotFound)
	assert.ErrorIs(t, movieNotFound, errNotFound)

	assert.NotErrorIs(t, fileNotFound, nil)
	assert.NotErrorIs(t, fileNotFound, cpuNotFound)
	assert.NotErrorIs(t, fileNotFound, movieNotFound)
	assert.NotErrorIs(t, fileNotFound, AlreadyExists)
	assert.NotErrorIs(t, fileNotFound, googleNotFound)

	assert.NotErrorIs(t, googleNotFound, fileNotFound)
	assert.NotErrorIs(t, googleNotFound, movieNotFound)

	assert.NotErrorIs(t, movieNotFound, googleNotFound)

	assert.ErrorIs(t, errors.WithMessage(fileNotFound, "message"), fileNotFound)
	assert.ErrorIs(t, errors.WithMessage(movieNotFound, "message"), fileNotFound)

	assert.NotErrorIs(t, errors.WithMessage(googleNotFound, "message"), fileNotFound)
}

func TestErrorMessage(t *testing.T) {
	{
		err := NotFound
		assert.Equal(t, "not found", err.Error())
	}

	{
		mine := NotFound.New("cannot load")
		assert.Equal(t, "cannot load", mine.Error())
	}
}

func TestCausedBy(t *testing.T) {
	{
		errInvalidAlgorithmF := func(alg string) RoxError {
			return InvalidArgs.New(fmt.Sprintf("invalid hashing algorithm %q used", alg))
		}
		assert.Equal(t, "invalid hashing algorithm \"SHA255\" used: only SHA256 is supported",
			errInvalidAlgorithmF("SHA255").CausedBy("only SHA256 is supported").Error())

		assert.ErrorIs(t, errInvalidAlgorithmF("SHA255"), InvalidArgs)
	}

	{
		assert.Equal(t, "not found: your fault",
			NotFound.CausedBy(errors.New("your fault")).Error())
	}

	{
		err := NotFound.New("lost forever").CausedBy("swallowed by Kraken")
		assert.ErrorIs(t, err, NotFound)
	}
}
