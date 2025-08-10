package phonehome

import (
	"fmt"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/eventual"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
)

var (
	log = logging.LoggerForModule()
)

const (
	// TenantIDLabel is the name of the k8s object label that holds the cloud
	// services tenant ID. The value of the label becomes the group ID if not empty.
	TenantIDLabel = "rhacs.redhat.com/tenant"

	storageKeyTimeout = time.Minute
)

// Config represents a telemetry client instance configuration.
type config struct {
	// storageKey should be eventually set by the client.
	// Any attempt to send telemetry data will wait for the key.
	storageKey *eventual.Value[string]
	// endpoint is the telemetry storage endpoint URL.
	endpoint string
	// configURL is the URL from where to download runtime configuration.
	configURL string
	// onReconfigure is called every time a remote configuration is downloaded.
	onReconfigure func(*RuntimeConfig)

	// clientID identifies an entity that reports telemetry data.
	clientID string
	// clientType tells what kind of client is sending data.
	clientType string
	// clientVersion is the client version.
	clientVersion string

	// groups is a map of group type to a list of group names.
	groups []telemeter.Option

	awaitInitialIdentity bool

	pushInterval time.Duration
	batchSize    int

	// gatherPeriod of identity gathering. Default is 1 hour.
	gatherPeriod time.Duration

	stateMux sync.RWMutex
}

func (c *config) String() string {
	if c == nil {
		return "<nil configuration>"
	}
	groups := telemeter.ApplyOptions(c.groups)
	return fmt.Sprintf(
		`endpoint: %q, key: %q, configURL: %q,`+
			` client ID: %q, client type: %q, client version: %q,`+
			` await initial identity: %v,`+
			` groups: %v, gathering period: %v,`+
			` batch size: %d, push interval: %v`,
		c.endpoint, c.storageKey, c.configURL,
		c.clientID, c.clientType, c.clientVersion,
		c.awaitInitialIdentity,
		groups.Groups, c.gatherPeriod,
		c.batchSize, c.pushInterval,
	)
}

func (c *config) withRLock(f func() bool) bool {
	return c != nil && concurrency.WithRLock1(&c.stateMux, func() bool {
		return f()
	})
}

func (c *config) withLock(f func() bool) bool {
	return c != nil && concurrency.WithLock1(&c.stateMux, func() bool {
		return f()
	})
}

type Option func(*config)

func applyOptions(opts []Option) *config {
	var cfg config
	for _, o := range opts {
		if o != nil {
			o(&cfg)
		}
	}
	return &cfg
}

// WithConnectionConfiguration sets the connection parameters.
func WithConnectionConfiguration(endpoint, storageKey, configURL string) Option {
	var key *eventual.Value[string]
	if storageKey == "" {
		key = eventual.New[string](eventual.WithTimeout(storageKeyTimeout),
			eventual.WithOnTimeout(func(set bool) {
				if set {
					log.Warn("timeout waiting for storage key")
				}
			}))
	} else {
		key = eventual.Now(storageKey)
	}
	return func(cfg *config) {
		cfg.endpoint = endpoint
		cfg.configURL = configURL
		cfg.storageKey = key
	}
}

// WithAwaitInitialIdentity makes the Track events wait until the initial
// identity is sent.
func WithAwaitInitialIdentity() Option {
	return func(cfg *config) {
		cfg.awaitInitialIdentity = true
	}
}

func WithConfigureCallback(callback func(*RuntimeConfig)) Option {
	return func(cfg *config) {
		cfg.onReconfigure = callback
	}
}

func WithGatheringPeriod(d time.Duration) Option {
	return func(cfg *config) {
		cfg.gatherPeriod = d
	}
}

func WithBatchSize(n int) Option {
	return func(cfg *config) {
		cfg.batchSize = n
	}
}

func WithPushInterval(d time.Duration) Option {
	return func(cfg *config) {
		cfg.pushInterval = d
	}
}

// WithClient allows for modifying the ClientID and ClientType call options.
func WithClient(clientID string, clientType, clientVersion string) Option {
	return func(cfg *config) {
		cfg.clientID = clientID
		cfg.clientType = clientType
		cfg.clientVersion = clientVersion
	}
}

// WithGroup appends a group for an event.
func WithGroup(groupType string, groupID string) Option {
	return func(cfg *config) {
		cfg.groups = append(cfg.groups, telemeter.WithGroup(groupType, groupID))
	}
}
