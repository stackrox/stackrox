package htpasswd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteAndRead(t *testing.T) {
	// Given an htpasswd file with some identities set...
	users := map[string]string{
		"user":  "pass",
		"other": "user",
		"admin": "password",
	}

	hf := New()
	for u, p := range users {
		require.NoError(t, hf.Set(u, p))
	}

	buf := bytes.NewBuffer([]byte{})
	err := hf.Write(buf)
	require.NoError(t, err)

	// ...when the file is read back...
	readHf, err := ReadHashFile(buf)
	require.NoError(t, err)

	// ...the right identities should work, and others should not.
	for u, p := range users {
		assert.True(t, hf.Check(u, p))
		assert.True(t, readHf.Check(u, p))
	}
	assert.False(t, hf.Check("user", "something else"))
	assert.False(t, readHf.Check("user", "something else"))
	assert.False(t, hf.Check("admin", ""))
	assert.False(t, readHf.Check("admin", ""))
}
