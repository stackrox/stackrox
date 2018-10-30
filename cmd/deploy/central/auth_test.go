package central

import (
	"bytes"
	"regexp"
	"testing"

	"github.com/stackrox/rox/pkg/auth/authproviders/userpass/htpasswd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPasswordGeneratedWhenEmpty(t *testing.T) {
	testHtpasswd(t, "")
}

func TestAssignedPassword(t *testing.T) {
	testHtpasswd(t, "testpass")
}

func testHtpasswd(t *testing.T, password string) {
	cfg := &Config{
		Password: password,
	}
	htpasswdFile, err := generateHtpasswd(cfg)
	require.NoError(t, err)

	assert.NotEmpty(t, cfg.Password)

	hf, err := htpasswd.ReadHashFile(bytes.NewBuffer(htpasswdFile))
	require.NoError(t, err)
	assert.True(t, hf.Check(adminUsername, cfg.Password))
}

func TestGeneratedPasswordIsAlphanumeric(t *testing.T) {
	const tries = 20
	for i := 0; i < tries; i++ {
		pw := createPassword()
		match, err := regexp.Match(`^[a-zA-Z0-9]{25}$`, []byte(pw))
		require.NoError(t, err)
		assert.Truef(t, match, "Password '%s' didn't match expected format", pw)
	}
}
