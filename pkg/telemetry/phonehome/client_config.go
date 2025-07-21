package phonehome

import (
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
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

	stateMux sync.RWMutex
}

// Reconfigure updates the configuration, potentially from the provided remote
// URL. defaultKey is returned within the RuntimeConfig if no better value is
// found. It will not update an inactive config.
func (c *Client) Reconfigure(cfgURL, defaultKey string) (*RuntimeConfig, error) {
	var err error
	var rc *RuntimeConfig
	var previouslyMissingKey bool

	if c == nil || !concurrency.WithLock1(&c.stateMux, func() bool {
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
		previouslyMissingKey = c.StorageKey == "" && c.enabled
		c.StorageKey = rc.Key
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

func (cfg *Config) GetStorageKey() string {
	if cfg == nil {
		return ""
	}
	return concurrency.WithRLock1(&cfg.stateMux, func() string {
		return cfg.StorageKey
	})
}

// IsActive tells whether telemetry configuration allows for data collection
// now or later. An inactive configuration cannot be reconfigured.
func (cfg *Config) IsActive() bool {
	return cfg != nil && concurrency.WithRLock1(&cfg.stateMux, func() bool { return cfg.isActiveNoLock() })
}

func (cfg *Config) isActiveNoLock() bool {
	return cfg.StorageKey != DisabledKey
}

// AddInterceptorFuncs appends the custom list of telemetry interceptors with
// the provided functions.
func (cfg *Config) AddInterceptorFuncs(event string, f ...Interceptor) {
	cfg.interceptorsLock.Lock()
	defer cfg.interceptorsLock.Unlock()
	if cfg.interceptors == nil {
		cfg.interceptors = make(map[string][]Interceptor, len(f))
	}
	cfg.interceptors[event] = append(cfg.interceptors[event], f...)
}
