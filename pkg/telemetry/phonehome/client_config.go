package phonehome

import (
	"context"
	"net/http"
	"time"

	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/segment"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()
)

// TenantIDLabel is the name of the k8s object label that holds the cloud
// services tenant ID. The value of the label becomes the group ID if not empty.
const TenantIDLabel = "rhacs.redhat.com/tenant"

// Interceptor is a function which will be called on every API call if none of
// the previous interceptors in the chain returned false.
// An Interceptor function may add custom properties to the props map so that
// they appear in the event.
type Interceptor func(rp *RequestParams, props map[string]any) bool

// Config represents a telemetry client instance configuration.
type Config struct {
	// ClientID identifies an entity that reports telemetry data.
	ClientID string
	// ClientName tells what kind of client is sending data.
	ClientName string
	// ClientVersion is the client version.
	ClientVersion string
	// GroupType identifies the main group type to which the client belongs.
	GroupType string
	// GroupID identifies the ID of the GroupType group.
	GroupID string

	StorageKey   string
	Endpoint     string
	PushInterval time.Duration
	BatchSize    int

	// The period of identity gathering. Default is 1 hour.
	GatherPeriod time.Duration

	telemeter telemeter.Telemeter
	gatherer  Gatherer

	onceTelemeter sync.Once
	onceGatherer  sync.Once

	// Map of event name to the list of interceptors, that gather properties for
	// the event.
	interceptors     map[string][]Interceptor
	interceptorsLock sync.RWMutex

	// enabled is an additional switch to enable or disable a well configured
	// client.
	enabled  bool
	stateMux sync.RWMutex
}

// Reconfigure updates the configuration, potentially from the provided remote
// URL. defaultKey is returned within the RuntimeConfig if no better value is
// found. It will not update an inactive config.
func (cfg *Config) Reconfigure(cfgURL, defaultKey string) (*RuntimeConfig, error) {
	var err error
	var rc *RuntimeConfig
	var previouslyMissingKey bool

	if !cfg.withLock(func() bool {
		if !cfg.isActiveNoLock() {
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
		previouslyMissingKey = cfg.StorageKey == "" && cfg.enabled
		cfg.StorageKey = rc.Key
		return true
	}) {
		return nil, err
	}

	if rc.Key == "" || rc.Key == DisabledKey {
		cfg.Disable()
	} else if previouslyMissingKey {
		cfg.Enable()
	}
	return rc, nil
}

func (cfg *Config) withLock(f func() bool) bool {
	if cfg == nil {
		return false
	}
	cfg.stateMux.Lock()
	defer cfg.stateMux.Unlock()
	return f()
}

func (cfg *Config) withRLock(f func() bool) bool {
	if cfg == nil {
		return false
	}
	cfg.stateMux.RLock()
	defer cfg.stateMux.RUnlock()
	return f()
}

// IsActive tells whether telemetry configuration allows for data collection
// now or later. An inactive configuration cannot be reconfigured.
func (cfg *Config) IsActive() bool {
	return cfg.withRLock(func() bool { return cfg.isActiveNoLock() })
}

func (cfg *Config) isActiveNoLock() bool {
	return cfg.StorageKey != DisabledKey
}

// IsEnabled tells whether the configuration allows for data collection now.
func (cfg *Config) IsEnabled() bool {
	return cfg.withRLock(func() bool { return cfg.isEnabledNoLock() })
}

func (cfg *Config) isEnabledNoLock() bool {
	return cfg.StorageKey != "" && cfg.StorageKey != DisabledKey && cfg.enabled
}

// Enable data reporting if the client is configured.
func (cfg *Config) Enable() {
	if !cfg.withLock(func() bool {
		if !cfg.isActiveNoLock() || cfg.isEnabledNoLock() {
			return false
		}
		cfg.enabled = true
		return true
	}) {
		return
	}
	cfg.Gatherer().Start(
		telemeter.WithGroups(cfg.GroupType, cfg.GroupID),
		// Don't capture the time, but call WithNoDuplicates on every gathering
		// iteration, so that the time is updated.
		func(co *telemeter.CallOptions) {
			// Issue a possible duplicate only once a day as a heartbeat.
			telemeter.WithNoDuplicates(time.Now().Format(time.DateOnly))(co)
		},
	)
}

// Disable data reporting of the configured client.
func (cfg *Config) Disable() {
	if !cfg.withLock(func() bool {
		if !cfg.isEnabledNoLock() {
			return false
		}
		cfg.enabled = false
		return true
	}) {
		return
	}
	cfg.Gatherer().Stop()
}

// Gatherer returns the telemetry gatherer instance.
func (cfg *Config) Gatherer() Gatherer {
	if !cfg.IsActive() {
		return &nilGatherer{}
	}
	cfg.onceGatherer.Do(func() {
		period := cfg.GatherPeriod
		if cfg.GatherPeriod.Nanoseconds() == 0 {
			period = 1 * time.Hour
		}
		if cfg.gatherer != nil {
			// cfg.gatherer could be set to a mock for testing purposes.
			return
		}
		// If configuration is disabled, cfg.Telemeter() returns nilTelemeter.
		_ = cfg.Telemeter()
		cfg.gatherer = newGatherer(cfg.ClientName, cfg.telemeter, period)
	})
	return cfg.gatherer
}

// Telemeter returns the instance of the telemeter.
func (cfg *Config) Telemeter() telemeter.Telemeter {
	if !cfg.IsActive() {
		return &nilTelemeter{}
	}
	cfg.onceTelemeter.Do(func() {
		if cfg.telemeter != nil {
			// cfg.telemeter could be set to a mock for testing purposes.
			return
		}
		cfg.telemeter = segment.NewTelemeter(
			cfg.StorageKey,
			cfg.Endpoint,
			cfg.ClientID,
			cfg.ClientName,
			cfg.ClientVersion,
			cfg.PushInterval,
			cfg.BatchSize)
	})
	if !cfg.IsEnabled() {
		return &nilTelemeter{}
	}
	return cfg.telemeter
}

// AddInterceptorFunc appends the custom list of telemetry interceptors with the
// provided function.
func (cfg *Config) AddInterceptorFunc(event string, f Interceptor) {
	cfg.interceptorsLock.Lock()
	defer cfg.interceptorsLock.Unlock()
	if cfg.interceptors == nil {
		cfg.interceptors = make(map[string][]Interceptor, 1)
	}
	cfg.interceptors[event] = append(cfg.interceptors[event], f)
}

// GetGRPCInterceptor returns an API interceptor function for GRPC requests.
func (cfg *Config) GetGRPCInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)
		rp := getGRPCRequestDetails(ctx, err, info.FullMethod, req)
		go cfg.track(rp)
		return resp, err
	}
}

// GetHTTPInterceptor returns an API interceptor function for HTTP requests.
func (cfg *Config) GetHTTPInterceptor() httputil.HTTPInterceptor {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			statusTrackingWriter := httputil.NewStatusTrackingWriter(w)
			handler.ServeHTTP(statusTrackingWriter, r)
			status := 0
			if sptr := statusTrackingWriter.GetStatusCode(); sptr != nil {
				status = *sptr
			}
			rp := getHTTPRequestDetails(r.Context(), r, status)
			go cfg.track(rp)
		})
	}
}
