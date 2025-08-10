package phonehome

import (
	"context"
	"fmt"
	"net/http"
	"time"

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

	// 53 minutes to not resonate with other periodic processes.
	reconfigurationPeriod = 53 * time.Minute
	consentTimeout        = time.Minute
)

// Client wraps telemetry configuration and implements some related methods.
type Client struct {
	config *config

	telemeter telemeter.Telemeter

	gatherer     Gatherer
	onceGatherer sync.Once

	// Map of event name to the list of interceptors, that gather properties for
	// the event.
	interceptors     map[string][]Interceptor
	interceptorsLock sync.RWMutex

	// consented is an additional switch to enable or disable a well configured
	// client.
	consented *eventual.Value[bool]

	// identified is checked on every Track call. This is to ensure the
	// group initializing events are sent before.
	identified chan struct{}
}

// noopClient is an inactive client, that cannot be activated.
func noopClient() *Client {
	return &Client{
		config:    &config{storageKey: eventual.Now(DisabledKey)},
		consented: eventual.Now(false),
	}
}

// NewClient returns a configured client instance.
// The returned client has to be eventually provided with the user consent
// status. Otherwise, it will be automatically deactivated after a timeout.
// At least WithClient and WithConnectionConfiguration have to be provided.
func NewClient(opts ...Option) *Client {
	if len(opts) == 0 {
		return noopClient()
	}

	cfg := applyOptions(opts)

	switch {
	case cfg.storageKey.IsSet() && cfg.storageKey.Get() == DisabledKey:
		return noopClient()
	// We want to avoid any reporting in non-production environments to not add
	// testing noise to the real self-managed telemetry data.
	// If no key is provided for a release binary version, the client will use a
	// hardcoded key for self-managed installations.
	// Therefore, for such a case, a no-op client is returned for non-release
	// builds.
	// For testing purposes, a key has to be set.
	// TODO(ROX-17726): update this comment when the key is no longer hardcoded.
	case !version.IsReleaseVersion() && (!cfg.storageKey.IsSet() || cfg.storageKey.Get() == ""):
		return noopClient()
	case version.IsReleaseVersion() && (!cfg.storageKey.IsSet() || cfg.storageKey.Get() == "") && cfg.configURL == "":
		return noopClient()
	default:
		c := newOperationalClient(cfg)
		if version.IsReleaseVersion() {
			go c.startPeriodicReload(reconfigurationPeriod)
		}
		return c
	}
}

// newOperationalClient returns a fully operational client.
// For testing convenience, this function won't start periodic reconfiguration.
func newOperationalClient(cfg *config) *Client {
	c := &Client{
		config: cfg,

		// enabled will be set to false after the timeout, if it is not set
		// explicitly. This is to unblock potentially blocked tracking
		// goroutines, waiting for the condition.
		consented: eventual.New[bool](eventual.WithTimeout(consentTimeout),
			eventual.WithOnTimeout(func(set bool) {
				if set {
					log.Warn("telemetry disabled" +
						" after timeout waiting for client consent status")
				}
			}),
		)}
	if cfg.awaitInitialIdentity {
		c.identified = make(chan struct{})
	}
	return c
}

// InitialIdentitySent confirms that the client identity has been sent, so
// Track events can be unblocked.
func (c *Client) InitialIdentitySent() {
	if c.identified != nil {
		select {
		case <-c.identified:
			// Already closed.
		default:
			close(c.identified)
		}
	}
}

func (c *Client) isIdentitySent() bool {
	select {
	case <-c.identified:
		return true
	default:
		return false
	}
}

func (c *Client) String() (s string) {
	_ = c.config.withRLock(func() bool {
		s = fmt.Sprintf("%v, consent: %s, identity sent: %v",
			c.config, c.consented, c.isIdentitySent())
		return true
	})
	return
}

func (c *Client) WithGroups() (opts []telemeter.Option) {
	_ = c.config.withRLock(func() bool {
		opts = c.config.groups
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

	if !c.config.withLock(func() bool {
		// Allow for reconfiguring a not yet configured client, which is
		// temporarily inactive as the key is not set.
		if c.config.storageKey.IsSet() && c.config.storageKey.Get() == DisabledKey {
			err = errox.InvalidArgs.New("telemetry is disabled")
			return false
		}
		rc, err = downloadConfig(c.config.configURL)
		if err != nil {
			return false
		}
		// We do not want to send test data accidentally, so we ignore the
		// non-DISABLED remote key in non-release environment.
		// But for testing purposes we keep the remote campaign.
		if !version.IsReleaseVersion() && rc.Key != DisabledKey {
			rc.Key = c.config.storageKey.Get()
		}
		// The key has changed, the telemeter needs to be reset.
		if c.config.storageKey.IsSet() && c.config.storageKey.Get() != rc.Key {
			if c.telemeter != nil {
				c.telemeter.Stop()
				c.telemeter = nil
			}
		}
		c.config.storageKey.Set(rc.Key)
		return true
	}) {
		return err
	}

	// The rc.Key could be empty, which tells the client to not send anything
	// until a new non-empty rc.Key is delivered.

	// Once remotely disabled, we won't be able to re-enable the client it
	// remotely.
	if rc.Key == DisabledKey {
		c.WithdrawConsent()
	} else if c.config.onReconfigure != nil {
		c.config.onReconfigure(rc)
	}
	return nil
}

// IsActive tells whether the configuration allows for data collection now.
// Warning: it will wait until the client has clarified the consent.
func (c *Client) IsActive() bool {
	return c.IsEnabled() &&
		c.config.storageKey.IsSet() &&
		c.config.storageKey.Get() != "" &&
		c.consented.Get() // This may wait.
}

// IsEnabled returns true if the client can be activated now or later.
func (c *Client) IsEnabled() bool {
	return c != nil && (c.config == nil ||
		(!c.config.storageKey.IsSet() ||
			c.config.storageKey.Get() != DisabledKey))
}

func (c *Client) HashUserID(userID string, authProviderID string) string {
	return c.config.HashUserID(userID, authProviderID)
}

func (c *Client) HashUserAuthID(id authn.Identity) string {
	return c.config.HashUserAuthID(id)
}

func (c *Client) GetStorageKey() string {
	return c.config.storageKey.Get()
}

func (c *Client) GetEndpoint() (endpoint string) {
	c.config.withRLock(func() bool {
		endpoint = c.config.endpoint
		return true
	})
	return
}

// GrantConsent data reporting if the client is configured.
func (c *Client) GrantConsent() {
	c.consented.Set(true)
}

// WithdrawConsent data reporting of the configured client.
func (c *Client) WithdrawConsent() {
	c.consented.Set(false)
}

// Gatherer returns the telemetry gatherer instance.
func (c *Client) Gatherer() Gatherer {
	if !c.IsEnabled() {
		return &nilGatherer{}
	}
	c.onceGatherer.Do(func() {
		period := c.config.gatherPeriod
		if c.config.gatherPeriod.Nanoseconds() == 0 {
			period = 1 * time.Hour
		}
		if c.gatherer != nil {
			// c.gatherer could be set to a mock for testing purposes.
			return
		}
		c.gatherer = newGatherer(c.config.clientType, c.Telemeter, period)
	})
	return c.gatherer
}

// Telemeter returns an instance created for the current storage key.
// A new instance is created if the key changes.
func (c *Client) Telemeter() telemeter.Telemeter {
	if !c.IsActive() {
		return &nilTelemeter{}
	}
	var t telemeter.Telemeter
	c.config.withLock(func() bool {
		if c.telemeter == nil {
			c.telemeter = segment.NewTelemeter(
				c.config.storageKey.Get(),
				c.config.endpoint,
				c.config.clientID,
				c.config.clientType,
				c.config.clientVersion,
				c.config.pushInterval,
				c.config.batchSize,
				c.identified,
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
			log.Warnf("failed to configure telemetry client from %q: %v", c.config.configURL, err)
		}
	}
	warn(c.Reconfigure())
	for range time.NewTicker(period).C {
		warn(c.Reconfigure())
	}
}

// Group is a shortcut to Telemeter().Group.
func (c *Client) Group(opts ...telemeter.Option) {
	c.Telemeter().Group(opts...)
}

// Track is a shortcut to Telemeter().Track.
func (c *Client) Track(event string, props map[string]any, opts ...telemeter.Option) {
	c.Telemeter().Track(event, props, opts...)
}
