package centralclient

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/sync"
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
		// c.IsEnabled() will wait until the client is enabled
		// or disabled explicitly.
		assert.Equal(t,
			`endpoint: "https://console.redhat.com/connections/api",`+
				` initial key: "non-empty", configURL: "hardcoded",`+
				` client ID: "test-id", client type: "Central", client version: "`+version.GetMainVersion()+`",`+
				` await initial identity: true,`+
				` groups: map[Tenant:[test-id]], gathering period: 0s,`+
				` batch size: 1, push interval: 10m0s,`+
				` effective key: non-empty, consent: <not set>, identity sent: false`,
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

func Test_getCentralDeploymentProperties(t *testing.T) {
	const devVersion = "4.4.1-dev"
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

func newMockServer() (chan map[string]any, *httptest.Server) {
	data := make(chan map[string]any, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		d := json.NewDecoder(r.Body)
		var message map[string][]map[string]any
		d.Decode(&message)
		for _, m := range message["batch"] {
			data <- m
		}
	}))
	return data, server
}

func Test_centralClient_flow(t *testing.T) {
	data, s := newMockServer()
	defer s.Close()
	defer close(data)

	t.Setenv(env.TelemetryStorageKey.EnvVar(), "test-key")
	t.Setenv(env.TelemetryEndpoint.EnvVar(), s.URL)

	c := newCentralClient("test-instance")

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.RegisterCentralClient(&grpc.Config{}, "basic")
	}()

	c.GrantConsent()

	assert.Equal(t, "group", (<-data)["type"]) // for the central client.
	assert.Equal(t, "group", (<-data)["type"]) // for the admin user.
	wg.Wait()
	go c.Enable()
	assert.Equal(t, "identify", (<-data)["type"]) // initial central identity.

	// Asynchronous Track events may arrive in any order.
	events := []any{
		(<-data)["event"], (<-data)["event"],
	}
	assert.ElementsMatch(t, []any{"Updated Central Identity", "Telemetry Enabled"}, events)

	assert.True(t, c.IsActive())
	go c.Disable()
	assert.Equal(t, "Telemetry Disabled", (<-data)["event"])
}
