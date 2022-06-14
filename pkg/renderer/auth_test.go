package renderer

import (
	"bytes"
	"regexp"
	"testing"

	"github.com/stackrox/stackrox/pkg/auth/htpasswd"
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
	htpasswdFile, err := GenerateHtpasswd(cfg)
	require.NoError(t, err)

	assert.NotEmpty(t, cfg.Password)

	hf, err := htpasswd.ReadHashFile(bytes.NewBuffer(htpasswdFile))
	require.NoError(t, err)
	assert.True(t, hf.Check(adminUsername, cfg.Password))
}

func TestGeneratedPasswordIsAlphanumeric(t *testing.T) {
	const tries = 20
	re := regexp.MustCompile(`^[a-zA-Z0-9]{25}$`)
	for i := 0; i < tries; i++ {
		pw := CreatePassword()
		match := re.Match([]byte(pw))
		assert.Truef(t, match, "Password '%s' didn't match expected format", pw)
	}
}
