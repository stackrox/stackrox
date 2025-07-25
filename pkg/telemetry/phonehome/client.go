package phonehome

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/segment"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
	"google.golang.org/grpc"
)

// Client wraps telemetry configuration and implements some related methods.
type Client struct {
	config   Config
	stateMux sync.RWMutex

	telemeter     telemeter.Telemeter
	onceTelemeter sync.Once

	gatherer     Gatherer
	onceGatherer sync.Once

	// Map of event name to the list of interceptors, that gather properties for
	// the event.
	interceptors     map[string][]Interceptor
	interceptorsLock sync.RWMutex

	// enabled is an additional switch to enable or disable a well configured
	// client.
	enabled bool
}

// NewClient returns a configured client instance.
func NewClient(cfg *Config) *Client {
	if cfg == nil {
		return &Client{}
	}
	return &Client{config: *cfg}
}

func (c *Client) String() (cfg string) {
	_ = c.lockedRead(func() bool {
		cfg = fmt.Sprintf("%+v", c.config)
		return true
	})
	return
}

func (c *Client) lockedRead(f func() bool) bool {
	return c != nil && concurrency.WithRLock1(&c.stateMux, func() bool {
		return f()
	})
}

func (c *Client) lockedWrite(f func() bool) bool {
	return c != nil && concurrency.WithLock1(&c.stateMux, func() bool {
		return f()
	})
}

func (c *Client) WithGroups() (o telemeter.Option) {
	_ = c.lockedRead(func() bool {
		o = telemeter.WithGroups(c.config.GroupType, c.config.GroupID)
		return true
	})
	return
}

// Reconfigure updates the configuration, potentially from the provided remote
// URL. defaultKey is returned within the RuntimeConfig if no better value is
// found. It will not update an inactive config.
func (c *Client) Reconfigure(cfgURL, defaultKey string) (*RuntimeConfig, error) {
	var err error
	var rc *RuntimeConfig
	var previouslyMissingKey bool

	if !c.lockedWrite(func() bool {
		if !c.isActiveNoLock() {
			return false
		}
		rc, err = getRuntimeConfig(cfgURL, defaultKey)
		if err != nil {
			return false
		}

		// This condition allows for the controlled start in main: the
		// configuration is not enabled on the instantiation, so only an
		// explicit call to cfg.Enable() will enable tracking and start
		// gatherers.
		previouslyMissingKey = c.config.StorageKey == "" && c.enabled
		c.config.StorageKey = rc.Key
		return true
	}) {
		return nil, err
	}

	if rc.Key == "" || rc.Key == DisabledKey {
		c.Disable()
	} else if previouslyMissingKey {
		c.Enable()
	}
	return rc, nil
}

// IsEnabled tells whether the configuration allows for data collection now.
func (c *Client) IsEnabled() bool {
	return c.lockedRead(func() bool {
		return c.isEnabledNoLock()
	})
}

func (c *Client) isEnabledNoLock() bool {
	return c.isActiveNoLock() && c.config.StorageKey != "" && c.enabled
}

func (c *Client) IsActive() bool {
	return c.lockedRead(func() bool {
		return c.isActiveNoLock()
	})
}

func (c *Client) isActiveNoLock() bool {
	return c.config.StorageKey != DisabledKey
}

func (c *Client) HashUserID(userID string, authProviderID string) string {
	return c.config.HashUserID(userID, authProviderID)
}

func (c *Client) HashUserAuthID(id authn.Identity) string {
	return c.config.HashUserAuthID(id)
}

func (c *Client) GetStorageKey() (key string) {
	c.lockedRead(func() bool {
		key = c.config.StorageKey
		return true
	})
	return
}

func (c *Client) GetEndpoint() (endpoint string) {
	c.lockedRead(func() bool {
		endpoint = c.config.Endpoint
		return true
	})
	return
}

// Enable data reporting if the client is configured.
func (c *Client) Enable() {
	if c.lockedWrite(func() bool {
		if !c.isActiveNoLock() || c.isEnabledNoLock() {
			return false
		}
		c.enabled = true
		return true
	}) {
		c.Gatherer().Start(
			c.WithGroups(),
			// Don't capture the time, but call WithNoDuplicates on every gathering
			// iteration, so that the time is updated.
			func(co *telemeter.CallOptions) {
				// Issue a possible duplicate only once a day as a heartbeat.
				telemeter.WithNoDuplicates(time.Now().Format(time.DateOnly))(co)
			},
		)
	}
}

// Disable data reporting of the configured client.
func (c *Client) Disable() {
	if c.lockedWrite(func() bool {
		if !c.isEnabledNoLock() {
			return false
		}
		c.enabled = false
		return true
	}) {
		c.Gatherer().Stop()
	}
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
			// cfg.gatherer could be set to a mock for testing purposes.
			return
		}
		// If configuration is disabled, cfg.Telemeter() returns nilTelemeter.
		_ = c.Telemeter()
		c.gatherer = newGatherer(c.config.ClientName, c.telemeter, period)
	})
	return c.gatherer
}

// Telemeter returns the instance of the telemeter.
func (c *Client) Telemeter() telemeter.Telemeter {
	if !c.IsActive() {
		return &nilTelemeter{}
	}
	c.onceTelemeter.Do(func() {
		if c.telemeter != nil {
			// cfg.telemeter could be set to a mock for testing purposes.
			return
		}
		c.telemeter = segment.NewTelemeter(
			c.config.StorageKey,
			c.config.Endpoint,
			c.config.ClientID,
			c.config.ClientName,
			c.config.ClientVersion,
			c.config.PushInterval,
			c.config.BatchSize)
	})
	if !c.IsEnabled() {
		return &nilTelemeter{}
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
