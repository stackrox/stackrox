package urlfmt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatURL(t *testing.T) {
	val, err := FormatURL("server.smtp:8080", false, false)
	assert.NoError(t, err)
	assert.Equal(t, "http://server.smtp:8080", val)

	val, err = FormatURL("server.smtp:8080", true, false)
	assert.NoError(t, err)
	assert.Equal(t, "https://server.smtp:8080", val)

	val, err = FormatURL("server.smtp:8080", true, true)
	assert.NoError(t, err)
	assert.Equal(t, "https://server.smtp:8080/", val)

	// Scrub final slash
	val, err = FormatURL("server.smtp:8080/", true, false)
	assert.NoError(t, err)
	assert.Equal(t, "https://server.smtp:8080", val)

	val, err = FormatURL("http://server.smtp:8080/////", true, false)
	assert.NoError(t, err)
	assert.Equal(t, "http://server.smtp:8080", val)
}

func TestGetServerFromURL(t *testing.T) {
	assert.Equal(t, "localhost", GetServerFromURL("https://localhost"))
	assert.Equal(t, "localhost", GetServerFromURL("http://localhost"))
	assert.Equal(t, "localhost:6060", GetServerFromURL("http://localhost:6060/v1"))
}
