package centralclient

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestSetProxyToCentralMetric(t *testing.T) {
	tests := map[string]struct {
		inputProxyHost string
		wantEnabled    float64
		wantInfoLabel  string
	}{
		"proxy host enables proxy metrics": {
			inputProxyHost: "proxy.example.com:3128",
			wantEnabled:    1,
			wantInfoLabel:  "proxy.example.com:3128",
		},
		"empty proxy host reports direct connection": {
			inputProxyHost: "",
			wantEnabled:    0,
			wantInfoLabel:  "direct",
		},
		"literal direct value is treated as a proxy label": {
			inputProxyHost: "direct",
			wantEnabled:    1,
			wantInfoLabel:  "direct",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			proxyToCentralEnabled.Set(-1)
			proxyToCentralInfo.Reset()
			t.Cleanup(func() {
				proxyToCentralEnabled.Set(0)
				proxyToCentralInfo.Reset()
			})

			setProxyToCentralMetric(tc.inputProxyHost)

			assert.Equal(t, tc.wantEnabled, testutil.ToFloat64(proxyToCentralEnabled))
			assert.Equal(t, 1.0, testutil.ToFloat64(proxyToCentralInfo.WithLabelValues(tc.wantInfoLabel)))
		})
	}
}
