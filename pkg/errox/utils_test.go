package errox

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestAs(t *testing.T) {
	assert.NotNil(t, As[*RoxError](NotFound))
	assert.NotNil(t, As[*RoxError](NotFound.New("file not found")))
	assert.NotNil(t, As[*RoxError](NotFound.CausedBy("test")))
	assert.NotNil(t, As[*RoxError](NotFound.CausedBy(errors.New("test"))))
	assert.NotNil(t, As[*RoxError](errors.Wrap(NotFound, "test")))
	assert.NotNil(t, As[error](NotFound))
	assert.Nil(t, As[*RoxError](errors.New("test")))
}
