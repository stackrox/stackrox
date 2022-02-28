package errox

import (
	"errors"
	"fmt"
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
}

func TestCausedBy(t *testing.T) {
	{
		errInvalidAlgorithm := func(alg string) RoxError {
			return InvalidArgs.New(fmt.Sprintf("invalid hashing algorithm %q used", alg))
		}
		assert.Equal(t, "invalid hashing algorithm \"SHA255\" used: only SHA256 is supported",
			errInvalidAlgorithm("SHA255").CausedBy("only SHA256 is supported").Error())
	}
	{
		assert.Equal(t, "not found: your fault",
			NotFound.CausedBy(errors.New("your fault")).Error())
	}
}
