package centralclient

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/segment/mock"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
)

func Test_newCentralClient(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		c := newCentralClient("test-id")
		// Configuration has an empty key, but not running a release version,
		// therefore the client is not active.
		assert.False(t, c.IsEnabled())
		// Telemetry should be disabled in test environment with no key provided.
		assert.False(t, c.IsActive())
	})

	t.Run("with a key in env", func(t *testing.T) {
		t.Setenv(env.TelemetryStorageKey.EnvVar(), "non-empty")
		c := newCentralClient("test-id")
		assert.True(t, c.IsEnabled())
		assert.Equal(t,
			`endpoint: "https://console.redhat.com/connections/api",`+
				` initial key: "non-empty", configURL: "hardcoded",`+
				` client ID: "test-id", client type: "Central", client version: "`+version.GetMainVersion()+`",`+
				` await initial identity: true,`+
				` groups: map[Tenant:[test-id]], gathering period: 0s,`+
				` batch size: 1, push interval: 10m0s,`+
				` effective key: non-empty, consent: <unset>, identity sent: false`,
			c.String())
	})

	t.Run("offline", func(t *testing.T) {
		t.Setenv(env.OfflineModeEnv.EnvVar(), "true")
		c := newCentralClient("test-id")
		assert.False(t, c.IsEnabled())
		assert.False(t, c.IsActive())
		assert.Equal(t, `endpoint: "", initial key: "", configURL: "",`+
			` client ID: "", client type: "", client version: "",`+
			` await initial identity: false,`+
			` groups: map[], gathering period: 0s,`+
			` batch size: 0, push interval: 0s,`+
			` effective key: DISABLED, consent: false, identity sent: false`,
			c.String())
	})
}

func Test_noopClient(t *testing.T) {
	assert.False(t, noopClient("").IsEnabled())
	assert.False(t, noopClient("id").IsEnabled())
}

func Test_getCentralDeploymentProperties(t *testing.T) {
	const devVersion = "4.4.1-dev"
	defer testutils.SetMainVersion(t, version.GetMainVersion())
	testutils.SetMainVersion(t, devVersion)
	t.Setenv(defaults.ImageFlavorEnvName, "opensource")

	props := getCentralDeploymentProperties()
	assert.Equal(t, map[string]any{
		"Central version":    "4.4.1-dev",
		"Chart version":      "400.4.1-dev",
		"Image Flavor":       "opensource",
		"Kubernetes version": "unknown",
		"Managed":            false,
		"Orchestrator":       "KUBERNETES_CLUSTER",
	}, props)
}

func printMessage(message map[string]any) string {
	var s strings.Builder
	for key, value := range mock.FilterMessageFields(message, "type", "event", "traits", "properties", "context") {
		if s.Len() > 0 {
			s.WriteString(", ")
		}
		s.WriteString(fmt.Sprintf("%s: %v", key, value))
	}
	id := message["messageId"].(string)
	if id[4] == '-' {
		s.WriteString(", prefixed message ID")
	}
	return s.String()
}

func Test_centralClient_flow(t *testing.T) {
	const devVersion = "4.4.1-dev"
	defer testutils.SetMainVersion(t, version.GetMainVersion())
	testutils.SetMainVersion(t, devVersion)

	s, data := mock.NewServer(1)
	defer s.Close()

	t.Setenv(env.TelemetryStorageKey.EnvVar(), "test-key")
	t.Setenv(env.TelemetryEndpoint.EnvVar(), s.URL)

	c := newCentralClient("test-instance")

	var gathered atomic.Bool
	c.Gatherer().AddGatherer(func(context.Context) (map[string]any, error) {
		gathered.Store(true)
		return map[string]any{
			"test": "value",
		}, nil
	})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.RegisterCentralClient(&grpc.Config{}, "basic")
	}()

	c.GrantConsent()

	// Adding central to the Tenant group:
	assert.Equal(t, `type: group, context: map[device:map[type:Central Server] groups:map[Tenant:[test-instance]]]`,
		printMessage(<-data))

	// Adding admin user to the same Tenant group:
	assert.Equal(t, "type: group, context: map[groups:map[Tenant:[test-instance]]]",
		printMessage(<-data))

	wg.Wait()
	go c.Enable()

	// Initial central identity with a prefixed message ID to drop duplicates.
	assert.Equal(t, `type: identify,`+
		` traits: map[`+
		`Central version:4.4.1-dev`+
		` Chart version:400.4.1-dev`+
		` Image Flavor:`+
		` Kubernetes version:unknown`+
		` Managed:false`+
		` Orchestrator:KUBERNETES_CLUSTER`+
		` test:value],`+
		` context: map[device:map[type:Central Server]],`+
		` prefixed message ID`,
		printMessage(<-data))

	assert.True(t, gathered.Load())

	// Asynchronous Track events may arrive in any order.
	events := []any{
		printMessage(<-data),
		printMessage(<-data),
	}
	assert.ElementsMatch(t, []any{
		`type: track, event: Updated Central Identity, context: map[device:map[type:Central Server]], prefixed message ID`,
		`type: track, event: Telemetry Enabled, context: map[device:map[type:Central Server]]`}, events)

	assert.True(t, c.IsActive())
	go c.Disable()
	assert.Equal(t, `type: track, event: Telemetry Disabled, context: map[device:map[type:Central Server]]`,
		printMessage(<-data))
}
