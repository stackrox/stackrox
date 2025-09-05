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
	storageKeyTimeout     = time.Minute
	consentTimeout        = time.Minute
)

// Client wraps telemetry configuration and implements some related methods.
type Client struct {
	// Immutable client configuration.
	config *config

	telemeter    telemeter.Telemeter
	telemeterMux sync.Mutex

	gatherer     Gatherer
	onceGatherer sync.Once

	// Map of event name to the list of interceptors, that gather properties for
	// the event.
	interceptors     map[string][]Interceptor
	interceptorsLock sync.RWMutex

	// Any attempt to send telemetry data will wait for the key, either set at
	// the moment of client initialization, or downloded from the configuration
	// server URL, or set to "" after storageKeyTimeout.
	storageKey eventual.Value[string]

	// consented is an additional switch to enable or disable a well configured
	// client.
	consented eventual.Value[bool]

	// identified is checked on every Track call. This is to ensure the
	// group and identity events are sent before.
	identified chan struct{}
}

// noopClient is an inactive client, that cannot be activated.
func noopClient() *Client {
	return &Client{
		config:     &config{},
		storageKey: eventual.Now(DisabledKey),
		consented:  eventual.Now(false),
	}
}

// NewClient returns a configured client instance.
// The returned client has to be eventually provided with the user consent
// status. Otherwise, it will be automatically deactivated after consentTimeout.
func NewClient(clientID, clientType, clientVersion string, opts ...Option) *Client {
	if clientID == "" || clientType == "" {
		return noopClient()
	}
	cfg := applyOptions(append(opts, withClient(clientID, clientType, clientVersion)))

	switch cfg.storageKey {
	case DisabledKey:
		return noopClient()
	case "":
		// We want to avoid any reporting in non-production environments to not
		// add testing noise to the real self-managed telemetry data.
		// If no key is provided for a release binary version, the client will
		// use a hardcoded key for self-managed installations.
		// Therefore, for such a case, a no-op client is returned for
		// non-release builds.
		// For testing purposes, a key has to be set.
		//
		// TODO(ROX-17726): update this comment when the key is no longer
		// hardcoded.
		if !version.IsReleaseVersion() || cfg.configURL == "" {
			return noopClient()
		}
	}
	c := newClientFromConfig(cfg)
	if version.IsReleaseVersion() {
		go c.startPeriodicReload(reconfigurationPeriod)
	}
	return c
}

// newClientFromConfig returns a fully operational client.
// For testing convenience, this function won't start periodic reconfiguration.
func newClientFromConfig(cfg *config) *Client {
	c := &Client{
		config: cfg,
		storageKey: eventual.New(eventual.WithType[string]().
			WithTimeout(storageKeyTimeout).
			WithContextCallback(func(_ context.Context) {
				log.Warn("timeout waiting for storage key")
			})),

		// enabled will be set to false after the timeout, if it is not set
		// explicitly. This is to unblock potentially blocked tracking
		// goroutines, waiting for the condition.
		consented: eventual.New(eventual.WithType[bool]().
			WithTimeout(consentTimeout).
			WithContextCallback(func(_ context.Context) {
				log.Warn("telemetry disabled" +
					" after timeout waiting for client consent status")
			}),
		)}
	if cfg.storageKey != "" {
		c.storageKey.Set(cfg.storageKey)
	}
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

func (c *Client) String() string {
	return fmt.Sprintf("%v, effective key: %v, consent: %s, identity sent: %v",
		c.config, c.storageKey, c.consented, c.isIdentitySent())
}

func (c *Client) WithGroups() []telemeter.Option {
	return c.config.groups
}

// Reconfigure updates the client's key from the provided URL and returns the
// remote configuration and an error.
// It will not update an inactive client.
func (c *Client) Reconfigure() error {
	// Allow for reconfiguring a not yet configured client, which is
	// temporarily inactive as the key is not set.
	if c.storageKey.IsSet() && c.storageKey.Get() == DisabledKey {
		return errox.InvalidArgs.New("telemetry is disabled")
	}
	rc, err := downloadConfig(c.config.configURL)
	if err != nil {
		return err
	}
	if c.storageKey.IsSet() && c.storageKey.Get() != rc.Key {
		// The key has changed, the telemeter needs to be reset.
		c.telemeterMux.Lock()
		defer c.telemeterMux.Unlock()
		if c.telemeter != nil {
			c.telemeter.Stop()
			c.telemeter = nil
		}
	}
	// We do not want to send test data accidentally, so we ignore the
	// non-DISABLED remote key in non-release environment.
	// But for testing purposes we keep the other fields.
	if version.IsReleaseVersion() || rc.Key == DisabledKey {
		c.storageKey.Set(rc.Key)
	}

	// The rc.Key could be empty, which tells the client to not send anything
	// until a new non-empty rc.Key is delivered.

	// Once remotely disabled, we won't be able to re-enable the client
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
		c.storageKey.IsSet() &&
		c.storageKey.Get() != "" &&
		c.consented.Get() // This may wait.
}

// IsEnabled returns true if the client can be activated now or later.
func (c *Client) IsEnabled() bool {
	if c == nil {
		return false
	}

	if !c.storageKey.IsSet() {
		// Storage key not currently set but can be later.
		return true
	}

	// This will not block because we know its already been set.
	return c.storageKey.Get() != DisabledKey
}

func (c *Client) HashUserID(userID string, authProviderID string) string {
	return c.config.HashUserID(userID, authProviderID)
}

func (c *Client) HashUserAuthID(id authn.Identity) string {
	return c.config.HashUserAuthID(id)
}

// GetStorageKey returns the storage key.
// May block until the key is set.
func (c *Client) GetStorageKey() string {
	return c.storageKey.Get()
}

func (c *Client) GetEndpoint() string {
	return c.config.endpoint
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
	c.telemeterMux.Lock()
	defer c.telemeterMux.Unlock()

	if c.telemeter == nil {
		c.telemeter = segment.NewTelemeter(
			c.storageKey.Get(),
			c.config.endpoint,
			c.config.clientID,
			c.config.clientType,
			c.config.clientVersion,
			c.config.pushInterval,
			c.config.batchSize,
			c.identified,
		)
	}
	return c.telemeter
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

// Identify is a shortcut to Telemeter().Identify().
func (c *Client) Identify(opts ...telemeter.Option) {
	c.Telemeter().Identify(opts...)
}

// Group is a shortcut to Telemeter().Group().
func (c *Client) Group(opts ...telemeter.Option) {
	c.Telemeter().Group(opts...)
}

// Track is a shortcut to Telemeter().Track().
func (c *Client) Track(event string, props map[string]any, opts ...telemeter.Option) {
	c.Telemeter().Track(event, props, opts...)
}
