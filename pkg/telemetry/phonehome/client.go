package phonehome

import (
	"context"
	"net/http"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/segment"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
	"google.golang.org/grpc"
)

// Client wraps telemetry configuration and implements some related methods.
type Client struct {
	*Config
	// enabled is an additional switch to enable or disable a well configured
	// client.
	enabled bool
}

// IsEnabled tells whether the configuration allows for data collection now.
func (c *Client) IsEnabled() bool {
	return c != nil && c.Config != nil && concurrency.WithRLock1(&c.stateMux, func() bool { return c.isEnabledNoLock() })
}

func (c *Client) isEnabledNoLock() bool {
	return c.StorageKey != "" && c.StorageKey != DisabledKey && c.enabled
}

// Enable data reporting if the client is configured.
func (c *Client) Enable() {
	if c == nil || !concurrency.WithLock1(&c.stateMux, func() bool {
		if !c.Config.isActiveNoLock() || c.isEnabledNoLock() {
			return false
		}
		c.enabled = true
		return true
	}) {
		return
	}
	c.Gatherer().Start(
		telemeter.WithGroups(c.GroupType, c.GroupID),
		// Don't capture the time, but call WithNoDuplicates on every gathering
		// iteration, so that the time is updated.
		func(co *telemeter.CallOptions) {
			// Issue a possible duplicate only once a day as a heartbeat.
			telemeter.WithNoDuplicates(time.Now().Format(time.DateOnly))(co)
		},
	)
}

// Disable data reporting of the configured client.
func (c *Client) Disable() {
	if c == nil || !concurrency.WithLock1(&c.stateMux, func() bool {
		if !c.isEnabledNoLock() {
			return false
		}
		c.enabled = false
		return true
	}) {
		return
	}
	c.Gatherer().Stop()
}

// Gatherer returns the telemetry gatherer instance.
func (c *Client) Gatherer() Gatherer {
	if !c.IsActive() {
		return &nilGatherer{}
	}
	c.onceGatherer.Do(func() {
		period := c.GatherPeriod
		if c.GatherPeriod.Nanoseconds() == 0 {
			period = 1 * time.Hour
		}
		if c.gatherer != nil {
			// cfg.gatherer could be set to a mock for testing purposes.
			return
		}
		// If configuration is disabled, cfg.Telemeter() returns nilTelemeter.
		_ = c.Telemeter()
		c.gatherer = newGatherer(c.ClientName, c.telemeter, period)
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
			c.StorageKey,
			c.Endpoint,
			c.ClientID,
			c.ClientName,
			c.ClientVersion,
			c.PushInterval,
			c.BatchSize)
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
