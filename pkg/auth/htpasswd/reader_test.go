package htpasswd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCannedHtpasswd(t *testing.T) {
	// To create additional realistic test cases, one can use the Apache
	// htpasswd utility:
	// https://httpd.apache.org/docs/2.4/programs/htpasswd.html
	// You must use bcrypt (-B).
	//#nosec G101 -- This is a false positive
	const htpasswd = `user:$2y$05$zOuqmZyoE82NGG4iitj91OrOQBoCrn0d/LiyHL833EvBzm0Wyy85.
other:$2y$05$b9mSdCSh6OnHhRDG/DAXee8USMpWYMK5XZcBZwFjQnCD5xQOu.F8y
admin:$2y$05$l.sGXGtYVWaoywFO06gDZeIHME8BFKWRuNv5PG4RLGUk0Yq/M4c86`
	users := map[string]string{
		"user":  "pass",
		"other": "user",
		"admin": "password",
	}

	// Given an htpasswd file with some identities set...
	buf := bytes.NewBufferString(htpasswd)
	hf, err := ReadHashFile(buf)
	require.NoError(t, err)

	// ...the right identities should work, and others should not.
	for u, p := range users {
		assert.True(t, hf.Check(u, p))
	}
	assert.False(t, hf.Check("user", "something else"))
	assert.False(t, hf.Check("admin", ""))
}
