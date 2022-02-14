package errox

import (
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

	fileNotFound := New(errNotFound, "file not found")
	cpuNotFound := New(errNotFound, "CPU not found")
	googleNotFound := New(errNotFound, "Google not found")
	movieNotFound := New(fileNotFound, "movie not found")

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
