package common

import (
	"crypto/tls"
	"net/http"
	"reflect"
	"testing"

	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func Test_newHTTPTransport_honorsProxyEnv(t *testing.T) {
	// Regression test for ROX-37149: roxctl ignored HTTP(S)_PROXY because the
	// hand-built transport left Proxy unset. Verify the transport delegates to
	// http.ProxyFromEnvironment for both HTTP/2 and forced HTTP/1 modes.
	for name, forceHTTP1 := range map[string]bool{
		"http2":      false,
		"forceHTTP1": true,
	} {
		t.Run(name, func(t *testing.T) {
			transport := newHTTPTransport(&tls.Config{}, forceHTTP1)
			require.NotNil(t, transport.Proxy, "transport must resolve the proxy from the environment")
			assert.Equal(t,
				reflect.ValueOf(http.ProxyFromEnvironment).Pointer(),
				reflect.ValueOf(transport.Proxy).Pointer(),
				"transport.Proxy must be http.ProxyFromEnvironment")
		})
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
