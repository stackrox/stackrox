package chaos

import (
	"context"
	"strconv"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/chaos/profile"
)

var (
	log = logging.LoggerForModule()
)

const (
	toxiproxyEndpoint     = "localhost:8474"
	toxiproxyCentralProxy = "localhost:8989"
)

// InitializeChaosConfiguration sets up Toxiproxy to forward requests to real central and control periodic disconnects.
func InitializeChaosConfiguration(ctx context.Context) {
	if !HasChaosProxy() || originalCentralEndpoint() == "" {
		log.Infof("Cannot start chaos proxy configuration (requires ROX_CHAOS_PROXY_ENABLED and ROX_CENTRAL_ENDPOINT_NO_PROXY set). Respectively: %s | %s",
			strconv.FormatBool(HasChaosProxy()), originalCentralEndpoint())
		return
	}

	log.Infof("Running sensor with Chaos Proxy enabled. This could produce disconnects between central and sensor. This should NEVER be enabled in production." +
		"If you see this message in production, make sure env ROX_CHAOS_PROXY_ENABLED is set to 'false'")

	client := toxiproxy.NewClient(toxiproxyEndpoint)
	proxy, err := client.CreateProxy("central", toxiproxyCentralProxy, originalCentralEndpoint())
	if err != nil {
		log.Warnf("Failed to start chaos client: %s", err)
		return
	}

	controller := profile.GetConfig(ctx, chaosProfile())
	go controller.Run(proxy)
}
