package tlsutils

import (
	"context"
	"crypto/tls"
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			require.Equal(t, 1, observedLogs.Len())
			errMessage := observedLogs.All()[0].ContextMap()["error"].(string)
			assert.True(t, strings.Contains(errMessage, "127.0.0.1") || strings.Contains(errMessage, "10001"))
		})
	}
}
