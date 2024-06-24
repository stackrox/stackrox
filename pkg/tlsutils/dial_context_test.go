package tlsutils

import (
	"context"
	"crypto/tls"
	"testing"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestDialContextError(t *testing.T) {
	tests := map[string]struct {
		addr    string
		message string
	}{
		"connection refused": {
			addr:    "127.0.0.1:10001",
			message: "unable to establish a TLS-enabled connection: dial tcp: connect"},
		"no port": {
			addr:    "127.0.0.1",
			message: "unable to establish a TLS-enabled connection: dial tcp: address: missing port in address"},
		"bad address": {
			addr:    "bad.svc:1",
			message: "unable to establish a TLS-enabled connection: dial tcp: lookup: no such host"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			observedZapCore, observedLogs := observer.New(zap.DebugLevel)
			log = &logging.LoggerImpl{
				InnerLogger: zap.New(observedZapCore).Sugar(),
			}

			_, err := DialContext(context.Background(), "tcp", test.addr, &tls.Config{})
			userMessage := errox.GetUserMessage(err)
			assert.Equal(t, test.message, userMessage)
			assert.Equal(t, 1, observedLogs.Len())
		})
	}
}
