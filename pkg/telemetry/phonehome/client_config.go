package phonehome

import (
	"context"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/segment"
	"google.golang.org/grpc"
)

var (
	onceGatherer  sync.Once
	onceTelemeter sync.Once

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
	// GroupID identifies the main group to which the client belongs.
	GroupID string

	StorageKey   string
	Endpoint     string
	PushInterval time.Duration

	// The period of identity gathering. Default is 1 hour.
	GatherPeriod time.Duration

	telemeter Telemeter
	gatherer  Gatherer

	// Map of event name to the list of interceptors, that gather properties for
	// the event.
	interceptors     map[string][]Interceptor
	interceptorsLock sync.Mutex
}

// Enabled tells whether telemetry data collection is enabled.
func (cfg *Config) Enabled() bool {
	return cfg != nil && cfg.StorageKey != ""
}

// Gatherer returns the telemetry gatherer instance.
func (cfg *Config) Gatherer() Gatherer {
	if cfg == nil {
		return &nilGatherer{}
	}
	onceGatherer.Do(func() {
		if cfg.Enabled() {
			period := cfg.GatherPeriod
			if cfg.GatherPeriod.Nanoseconds() == 0 {
				period = 1 * time.Hour
			}
			cfg.gatherer = newGatherer(cfg.ClientID, cfg.Telemeter(), period)
		} else {
			cfg.gatherer = &nilGatherer{}
		}
	})
	return cfg.gatherer
}

// Telemeter returns the instance of the telemeter.
func (cfg *Config) Telemeter() Telemeter {
	if cfg == nil {
		return &nilTelemeter{}
	}
	onceTelemeter.Do(func() {
		if cfg.Enabled() {
			cfg.telemeter = segment.NewTelemeter(
				cfg.StorageKey,
				cfg.Endpoint,
				cfg.ClientID,
				cfg.ClientName,
				cfg.PushInterval)
		} else {
			cfg.telemeter = &nilTelemeter{}
		}
	})
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
		rp := getGRPCRequestDetails(ctx, err, info, req)
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
			rp := getHTTPRequestDetails(r.Context(), r, statusCodeToError(statusTrackingWriter.GetStatusCode()))
			go cfg.track(rp)
		})
	}
}

func statusCodeToError(code *int) error {
	if code == nil || *code == http.StatusOK {
		return nil
	}
	return errors.Errorf("%d %s", *code, http.StatusText(*code))
}
