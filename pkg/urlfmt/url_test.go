package urlfmt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatURL(t *testing.T) {
	val := FormatURL("server.smtp:8080", NONE, NoTrailingSlash)
	assert.Equal(t, "server.smtp:8080", val)

	val = FormatURL("http://server.smtp:8080", NONE, NoTrailingSlash)
	assert.Equal(t, "server.smtp:8080", val)

	val = FormatURL("https://server.smtp:8080", NONE, NoTrailingSlash)
	assert.Equal(t, "server.smtp:8080", val)

	val = FormatURL("server.smtp:8080", InsecureHTTP, NoTrailingSlash)
	assert.Equal(t, "http://server.smtp:8080", val)

	val = FormatURL("server.smtp:8080", HTTPS, NoTrailingSlash)
	assert.Equal(t, "https://server.smtp:8080", val)

	val = FormatURL("server.smtp:8080", HTTPS, TrailingSlash)
	assert.Equal(t, "https://server.smtp:8080/", val)

	// Scrub final slash
	val = FormatURL("server.smtp:8080/", HTTPS, NoTrailingSlash)
	assert.Equal(t, "https://server.smtp:8080", val)

	val = FormatURL("http://server.smtp:8080/////", HTTPS, NoTrailingSlash)
	assert.Equal(t, "http://server.smtp:8080", val)
}

func TestGetServerFromURL(t *testing.T) {
	assert.Equal(t, "localhost", GetServerFromURL("https://localhost"))
	assert.Equal(t, "localhost", GetServerFromURL("http://localhost"))
	assert.Equal(t, "localhost:6060", GetServerFromURL("http://localhost:6060/v1"))
}

func TestTrimHTTPPrefixes(t *testing.T) {
	assert.Equal(t, "localhost", TrimHTTPPrefixes("https://localhost"))
	assert.Equal(t, "localhost", TrimHTTPPrefixes("http://localhost"))
	assert.Equal(t, "tcp://localhost", TrimHTTPPrefixes("tcp://localhost"))

	assert.Equal(t, "httpslocalhost", TrimHTTPPrefixes("httpslocalhost"))
	assert.Equal(t, " localhost", TrimHTTPPrefixes("https:// localhost"))
	assert.Equal(t, "localhost ", TrimHTTPPrefixes("https://localhost "))
}
