package safe

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunE(t *testing.T) {
	called := 0
	// Normal operation
	err := runE(func() error {
		called++
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, called)

	// Error operation
	someErr := errors.New("some error")
	err = runE(func() error {
		called++
		return someErr
	})
	assert.Equal(t, someErr, err)
	assert.Equal(t, 2, called)

	// Panic with a non-error
	err = runE(func() error {
		called++
		panic("oh noes")
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "caught panic")
	assert.Contains(t, err.Error(), "oh noes")
	assert.Equal(t, 3, called)

	// Panic with an error
	err = runE(func() error {
		called++
		panic(someErr)
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "caught panic")
	assert.True(t, errors.Is(err, someErr))
	assert.Equal(t, 4, called)
}
