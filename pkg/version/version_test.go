package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseCurrentVersion(t *testing.T) {
	_, err := parseMainVersion(GetMainVersion())
	assert.NoError(t, err)
}

func TestGetMajorMinor(t *testing.T) {
	assert.Equal(t, "3.73", GetMajorMinor("3.73.x"))
	assert.Equal(t, "38.753", GetMajorMinor("38.753.15"))
	assert.Equal(t, "38.753", GetMajorMinor("38.753.15.18"))
	assert.Equal(t, "38.75", GetMajorMinor("38.75"))
	assert.Equal(t, "38", GetMajorMinor("38"))
	assert.Equal(t, "a.b", GetMajorMinor("a.b.c"))
	assert.Equal(t, "a.b", GetMajorMinor("a.b.c-d.e"))
	assert.Equal(t, "", GetMajorMinor(""))
}
