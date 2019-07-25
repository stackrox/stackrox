package ioutils

import (
	"io"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestErrorReader_SomeError(t *testing.T) {
	t.Parallel()

	readErr := errors.New("some error during read")

	var buf [1]byte
	r := ErrorReader(readErr)
	n, err := r.Read(buf[:])
	assert.Zero(t, n)
	assert.Equal(t, readErr, err)
}

func TestErrorReader_NilError(t *testing.T) {
	t.Parallel()

	var buf [1]byte
	r := ErrorReader(nil)
	n, err := r.Read(buf[:])
	assert.Zero(t, n)
	assert.NotNil(t, err)
	assert.Equal(t, io.EOF, err)
}
