package errox

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	googleNotFound := errNotFound.Newf("G%sgle not found", "oo")
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
		errInvalidAlgorithmF := func(alg string) Error {
			return InvalidArgs.Newf("invalid hashing algorithm %q used", alg)
		}
		assert.Equal(t, "invalid hashing algorithm \"SHA255\" used: only SHA256 is supported",
			errInvalidAlgorithmF("SHA255").CausedBy("only SHA256 is supported").Error())

		assert.ErrorIs(t, errInvalidAlgorithmF("SHA255"), InvalidArgs)
	}

	{
		cause := errors.New("your fault")
		err := NotFound.CausedBy(cause)
		assert.Equal(t, "not found: your fault", err.Error())
		assert.ErrorIs(t, err, NotFound)
		assert.NotErrorIs(t, err, cause)
	}

	{
		err := NotFound.New("lost forever").CausedBy("swallowed by Kraken")
		assert.ErrorIs(t, err, NotFound)
	}

	{
		err := NotFound.New("absolute disaster").CausedByf("out of %v", "sense")
		assert.Equal(t, "absolute disaster: out of sense", err.Error())
		assert.ErrorIs(t, err, NotFound)
	}
}

func TestSentinelMessage(t *testing.T) {
	tests := []struct {
		err              error
		expectedSentinel error
	}{
		{NotFound, NotFound},
		{NotFound.CausedBy(InvalidArgs), NotFound},
		{NotFound.CausedBy("secret"), NotFound},
		{errors.WithMessage(NotFound, "secret"), NotFound},
		{errors.WithMessage(NotFound.New("secret"), "secret"), NotFound},
		{NotFound.New("secret"), NotFound},
		{errors.New("abc"), ServerError},
	}

	for _, test := range tests {
		re := GetBaseSentinelError(test.err)
		require.NotNil(t, re)
		assert.ErrorIs(t, re, test.expectedSentinel, re)
	}
}

func TestGetBaseSentinelError(t *testing.T) {
	assert.Equal(t, "not found", GetBaseSentinelError(
		NotFound.New("abc").New("def")).Error())
	assert.Equal(t, "not found", GetBaseSentinelError(
		errors.WithMessage(NotFound.New("abc").New("def"), "message")).Error())
}
