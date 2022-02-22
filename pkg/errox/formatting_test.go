package errox

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorMessage(t *testing.T) {
	{
		err := NotFound
		assert.Equal(t, "not found", err.Error())
	}

	{
		mine := NotFound.New("cannot load")
		assert.Equal(t, "cannot load", mine.Error())
	}

	{
		err := InvalidArgs.Template("custom {{.}}")("message")
		assert.Equal(t, "custom message", err.Error())
		assert.ErrorIs(t, err, InvalidArgs)
	}
}

func TestNewf(t *testing.T) {
	errFileNotFound := NotFound.Template("file '{{.}}' not found")
	assert.Equal(t, "file 'filename' not found", errFileNotFound("filename").Error())
}

func TestCausedBy(t *testing.T) {
	{
		errInvalidAlgorithm := InvalidArgs.Template("invalid hashing algorithm \"{{.}}\" used")
		assert.Equal(t, "invalid hashing algorithm \"SHA255\" used: only SHA256 is supported",
			errInvalidAlgorithm("SHA255").CausedBy("only SHA256 is supported").Error())
	}
	{
		assert.Equal(t, "not found: your fault",
			NotFound.CausedBy(errors.New("your fault")).Error())
	}
}
