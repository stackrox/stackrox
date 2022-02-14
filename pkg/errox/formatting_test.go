package errox

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
		err := Newf(InvalidArgs, "custom %s")("message")
		assert.Equal(t, "custom message", err.Error())
		assert.ErrorIs(t, err, InvalidArgs)
	}

	{
		err := New(os.ErrClosed, "not open")
		assert.Equal(t, "not open", err.Error())
		assert.ErrorIs(t, err, os.ErrClosed)
	}
}

func TestNewf(t *testing.T) {
	errFileNotFound := NotFound.Newf("file %q not found")
	assert.Equal(t, "file \"filename\" not found", errFileNotFound("filename").Error())
}

func TestExplain(t *testing.T) {
	{
		err := makeSentinel("test")
		assert.Equal(t, "test, explained", err.Explain("explained").Error())
	}
	{
		errInvalidAlgorithm := Newf(InvalidArgs, "invalid hashing algorithm %q used")
		assert.Equal(t, "invalid hashing algorithm \"SHA255\" used, only SHA256 is supported",
			errInvalidAlgorithm("SHA255").Explain("only SHA256 is supported").Error())
	}
	{
		assert.Equal(t, "file already closed, Mercury retrograde",
			Explain(os.ErrClosed, "Mercury retrograde").Error())
	}
}
