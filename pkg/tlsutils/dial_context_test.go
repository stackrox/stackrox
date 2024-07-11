package tlsutils

import (
	"context"
	"crypto/tls"
	"testing"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestDialContextError(t *testing.T) {
	tests := map[string]string{
		"connection refused": "127.0.0.1:10001",
		"no port":            "127.0.0.1",
		"bad address":        "127.0.0.1:10001/32",
	}

	for name, dialAddr := range tests {
		t.Run(name, func(t *testing.T) {
			observedZapCore, observedLogs := observer.New(zap.DebugLevel)
			log = &logging.LoggerImpl{
				InnerLogger: zap.New(observedZapCore).Sugar(),
			}

			_, err := DialContext(context.Background(), "tcp", dialAddr, &tls.Config{})

			assert.NotContains(t, err.Error(), "127.0.0.1")
			assert.NotContains(t, err.Error(), "10001")
			assert.Equal(t, 1, observedLogs.Len())
		})
	}
}
