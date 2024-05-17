package common

import (
	"net/http"
	"testing"

	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"github.com/stretchr/testify/assert"
)

func TestSanitizeHeaderValue(t *testing.T) {
	const vchars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"
	for value, expected := range map[string]string{
		"--abc ***":                 "--abc ***",
		"abc\ndef\tghi\rjkl\x00mno": "abc def ghi jkl mno",
		"":                          "",
		vchars:                      vchars,
	} {
		assert.Equal(t, expected, sanitizeHeaderValue(value), value)
	}
}

func Test_setCustomHeaders(t *testing.T) {
	headers := http.Header{}
	setCustomHeaders(phonehome.Headers(headers).Set)
	assert.Len(t, headers, 2)
	assert.Equal(t, "", headers.Get(clientconn.RoxctlCommandHeader))
	assert.Equal(t, "1", headers.Get(clientconn.RoxctlCommandIndexHeader))

	t.Setenv(env.ExecutionEnvironment.EnvVar(), "test")
	RoxctlCommand = "custom command"
	setCustomHeaders(phonehome.Headers(headers).Set)
	assert.Len(t, headers, 3)
	assert.Equal(t, RoxctlCommand, headers.Get(clientconn.RoxctlCommandHeader))
	assert.Equal(t, "2", headers.Get(clientconn.RoxctlCommandIndexHeader))
	assert.Equal(t, "test", headers.Get(clientconn.ExecutionEnvironment))
}
