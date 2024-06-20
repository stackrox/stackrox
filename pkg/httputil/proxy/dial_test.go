package proxy

import (
	"net/url"
	"testing"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stretchr/testify/assert"
)

func TestDialError(t *testing.T) {
	proxyURL := &url.URL{
		Host: "",
	}
	_, _, _, err := netutil.ParseEndpoint(proxyURL.Host)
	err = errox.NewSensitive(
		errox.WithPublicError(err),
		errox.WithSensitivef("invalid endpoint in proxy URL %q", proxyURL),
		errox.WithPublicMessage("invalid endpoint in proxy URL"),
	)
	assert.Equal(t, "invalid endpoint in proxy URL: empty endpoint specified", err.Error())
	assert.Equal(t, "invalid endpoint in proxy URL \"\": empty endpoint specified", errox.UnconcealSensitive(err))
}
