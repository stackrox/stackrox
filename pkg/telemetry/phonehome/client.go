package phonehome

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/eventual"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/segment"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
	"github.com/stackrox/rox/pkg/version"
	"google.golang.org/grpc"
)

const (
	// DisabledKey is a key value which disables the telemetry collection.
	// If the current key is DisabledKey, it won't be reconfigured.
	DisabledKey = "DISABLED"
)

// Client wraps telemetry configuration and implements some related methods.
type Client struct {
	config   Config
	stateMux sync.RWMutex

	telemeter telemeter.Telemeter

	gatherer     Gatherer
	onceGatherer sync.Once

	// Map of event name to the list of interceptors, that gather properties for
	// the event.
	interceptors     map[string][]Interceptor
	interceptorsLock sync.RWMutex

	// enabled is an additional switch to enable or disable a well configured
	// client.
	enabled *eventual.Value[bool]
}

// noopClient is an inactive client, that cannot be activated.
var noopClient = &Client{
	config:  Config{StorageKey: eventual.Now(DisabledKey)},
	enabled: eventual.Now(false),
}

// NewClient returns a configured client instance.
// The returned client has to be eventually enabled or disabled, according to
// the opt-in/out status. Otherwise, it will be automatically disabled after a
// timeout.
func NewClient(cfg *Config) *Client {
	if cfg == nil {
		return noopClient
	}

	if cfg.StorageKey == nil {
		cfg.StorageKey = eventual.New[string](eventual.WithTimeout(time.Minute),
			eventual.WithOnTimeout(func(set bool) {
				if set {
					log.Warn("timeout waiting for storage key")
				}
			}))
	}

	switch {
	case cfg.StorageKey.IsSet() && cfg.StorageKey.Get() == DisabledKey:
		return noopClient
	// We want to avoid any reporting in non-production environments to not add
	// testing noise to the real self-managed telemetry data.
	// If no key is provided for a release binary version, the client will use a
	// hardcoded key for self-managed installations.
	// Therefore, for such a case, a no-op client is returned for non-release
	// builds.
	// For testing purposes, a key has to be set.
	// TODO(ROX-17726): update this comment when the key is no longer hardcoded.
	case !version.IsReleaseVersion() && (!cfg.StorageKey.IsSet() || cfg.StorageKey.Get() == ""):
		return noopClient
	default:
		c := newOperationalClient(cfg)
		if c.config.ConfigURL != "" {
			// 53 minutes to not resonate with other periodic processes.
			go c.startPeriodicReload(53 * time.Minute)
		}
		return c
	}
}

// newOperationalClient returns a fully operational client.
// For testing convenience, this function won't start periodic reconfiguration.
func newOperationalClient(cfg *Config) *Client {
	return &Client{config: *cfg,
		// enabled will be set to false after the timeout, if it is not set
		// explicitly. This is to unblock potentially blocked tracking
		// goroutines, waiting for the condition.
		enabled: eventual.New[bool](eventual.WithTimeout(time.Minute),
			eventual.WithOnTimeout(func(set bool) {
				if set {
					log.Warn("telemetry disabled" +
						" after timeout waiting for client consent status")
				}
			}),
		)}
}

func (c *Client) String() (cfg string) {
	_ = c.withConfigRLock(func() bool {
		cfg = fmt.Sprintf("%+v", c.config)
		return true
	})
	return
}

func (c *Client) withConfigRLock(f func() bool) bool {
	return c != nil && concurrency.WithRLock1(&c.stateMux, func() bool {
		return f()
	})
}

func (c *Client) withConfigLock(f func() bool) bool {
	return c != nil && concurrency.WithLock1(&c.stateMux, func() bool {
		return f()
	})
}

func (c *Client) SetIDs(clientID, groupType, groupID string) {
	_ = c.withConfigLock(func() bool {
		c.config.ClientID = clientID
		c.config.GroupType = groupType
		c.config.GroupID = groupID
		return true
	})
}

func (c *Client) WithGroups() (o telemeter.Option) {
	_ = c.withConfigRLock(func() bool {
		o = telemeter.WithGroups(c.config.GroupType, c.config.GroupID)
		return true
	})
	return
}

// Reconfigure updates the client's key from the provided URL and returns the
// remote configuration and an error.
// It will not update an inactive client.
func (c *Client) Reconfigure() error {
	var err error
	var rc *RuntimeConfig

	if !c.withConfigLock(func() bool {
		// Allow for reconfiguring a not yet configured client, which is
		// temporarily inactive as the key is not set.
		if c.config.StorageKey.IsSet() && c.config.StorageKey.Get() == DisabledKey {
			err = errox.InvalidArgs.New("telemetry is disabled")
			return false
		}
		rc, err = downloadConfig(c.config.ConfigURL)
		if err != nil {
			return false
		}
		// We do not want to send test data accidentally, so we ignore the
		// non-DISABLED remote key in non-release environment.
		// But for testing purposes we keep the remote campaign.
		if !version.IsReleaseVersion() && rc.Key != DisabledKey {
			rc.Key = c.config.StorageKey.Get()
		}
		// The key has changed, the telemeter needs to be reset.
		if c.config.StorageKey.IsSet() && c.config.StorageKey.Get() != rc.Key {
			if c.telemeter != nil {
				c.telemeter.Stop()
				c.telemeter = nil
			}
		}
		c.config.StorageKey.Set(rc.Key)
		return true
	}) {
		return err
	}

	// The rc.Key could be empty, which tells the client to not send anything
	// until a new non-empty rc.Key is delivered.

	// Once remotely disabled, we won't be able to re-enable the client it
	// remotely.
	if rc.Key == DisabledKey {
		c.Disable()
	} else if c.config.OnReconfigure != nil {
		c.config.OnReconfigure(rc)
	}
	return nil
}

// IsEnabled tells whether the configuration allows for data collection now.
// Warning: it will wait until the client is explicitly enabled or disabled.
func (c *Client) IsEnabled() bool {
	return c.IsActive() &&
		c.config.StorageKey.IsSet() &&
		c.config.StorageKey.Get() != "" &&
		c.enabled.Get() // This may wait until the client is enabled.
}

// IsActive returns true if the client can be enabled now or later.
func (c *Client) IsActive() bool {
	return c != nil &&
		(!c.config.StorageKey.IsSet() ||
			c.config.StorageKey.Get() != DisabledKey)
}

func (c *Client) HashUserID(userID string, authProviderID string) string {
	return c.config.HashUserID(userID, authProviderID)
}

func (c *Client) HashUserAuthID(id authn.Identity) string {
	return c.config.HashUserAuthID(id)
}

func (c *Client) GetStorageKey() string {
	return c.config.StorageKey.Get()
}

func (c *Client) GetEndpoint() (endpoint string) {
	c.withConfigRLock(func() bool {
		endpoint = c.config.Endpoint
		return true
	})
	return
}

// Enable data reporting if the client is configured.
func (c *Client) Enable() {
	if !c.IsActive() {
		return
	}
	c.enabled.Set(true)
}

// Disable data reporting of the configured client.
func (c *Client) Disable() {
	c.enabled.Set(false)
}

// Gatherer returns the telemetry gatherer instance.
func (c *Client) Gatherer() Gatherer {
	if !c.IsActive() {
		return &nilGatherer{}
	}
	c.onceGatherer.Do(func() {
		period := c.config.GatherPeriod
		if c.config.GatherPeriod.Nanoseconds() == 0 {
			period = 1 * time.Hour
		}
		if c.gatherer != nil {
			// c.gatherer could be set to a mock for testing purposes.
			return
		}
		c.gatherer = newGatherer(c.config.ClientName, c.Telemeter, period, c.config.Identified)
	})
	return c.gatherer
}

// Telemeter returns an instance created for the current storage key.
// A new instance is created if the key changes.
func (c *Client) Telemeter() telemeter.Telemeter {
	if !c.IsEnabled() {
		return &nilTelemeter{}
	}
	var t telemeter.Telemeter
	c.withConfigLock(func() bool {
		if c.telemeter == nil {
			c.telemeter = segment.NewTelemeter(
				c.config.StorageKey.Get(),
				c.config.Endpoint,
				c.config.ClientID,
				c.config.ClientName,
				c.config.ClientVersion,
				c.config.PushInterval,
				c.config.BatchSize,
				c.config.Identified,
			)
		}
		t = c.telemeter
		return true
	})
	return t
}

// GetGRPCInterceptor returns an API interceptor function for GRPC requests.
func (c *Client) GetGRPCInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)
		rp := getGRPCRequestDetails(ctx, err, info.FullMethod, req)
		go c.track(rp)
		return resp, err
	}
}

// GetHTTPInterceptor returns an API interceptor function for HTTP requests.
func (c *Client) GetHTTPInterceptor() httputil.HTTPInterceptor {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			statusTrackingWriter := httputil.NewStatusTrackingWriter(w)
			handler.ServeHTTP(statusTrackingWriter, r)
			status := 0
			if sptr := statusTrackingWriter.GetStatusCode(); sptr != nil {
				status = *sptr
			}
			rp := getHTTPRequestDetails(r.Context(), r, status)
			go c.track(rp)
		})
	}
}

// AddInterceptorFuncs appends the custom list of telemetry interceptors with
// the provided functions.
func (c *Client) AddInterceptorFuncs(event string, f ...Interceptor) {
	c.interceptorsLock.Lock()
	defer c.interceptorsLock.Unlock()
	if c.interceptors == nil {
		c.interceptors = make(map[string][]Interceptor, len(f))
	}
	c.interceptors[event] = append(c.interceptors[event], f...)
}

// startPeriodicReload reloads and applies the configuration from the remote
// endpoint and starts a loop that does the same with the given period.
func (c *Client) startPeriodicReload(period time.Duration) {
	warn := func(err error) {
		if err != nil {
			log.Warnf("failed to configure telemetry client from %q: %v", c.config.ConfigURL, err)
		}
	}
	warn(c.Reconfigure())
	for range time.NewTicker(period).C {
		warn(c.Reconfigure())
	}
}
