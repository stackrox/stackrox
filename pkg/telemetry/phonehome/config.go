package phonehome

import (
	"fmt"
	"time"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
)

var (
	log = logging.LoggerForModule()
)

const (
	// TenantIDLabel is the name of the k8s object label that holds the cloud
	// services tenant ID. The value of the label becomes the group ID if not empty.
	TenantIDLabel = "rhacs.redhat.com/tenant"
)

// config stores internal client configuration.
type config struct {
	// storageKey, if not empty, sets the storage key on client initialization.
	storageKey string
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

	// awaitInitialIdentity tells whether Track calls must wait until the client
	// confirms that all initial identity and groups data is sent.
	awaitInitialIdentity bool

	// gatherPeriod of identity gathering. Default is 1 hour.
	gatherPeriod time.Duration

	// The maximum number of messages that will be sent in one API call.
	// Messages will be sent when they've been queued up to the maximum batch
	// size or when the flushing interval timer triggers.
	// Note that the API will still enforce a 500KB limit on each HTTP request
	// which is independent from the number of embedded messages.
	// If batchSize is 1, the events are sent synchronously.
	batchSize int

	// The flushing interval of the client. Messages will be sent when they've
	// been queued up to the maximum batch size or when the flushing interval
	// timer triggers.
	pushInterval time.Duration
}

func (c *config) String() string {
	if c == nil {
		return "<nil configuration>"
	}
	groups := telemeter.ApplyOptions(c.groups)
	return fmt.Sprintf(
		`endpoint: %q, initial key: %q, configURL: %q,`+
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

// WithEndpoint sets the custom storage endpoint.
func WithEndpoint(endpoint string) Option {
	return func(cfg *config) {
		cfg.endpoint = endpoint
	}
}

// WithStorageKey sets the storage key.
func WithStorageKey(key string) Option {
	return func(cfg *config) {
		cfg.storageKey = key
	}
}

// WithConfigURL sets the configuration server URL.
func WithConfigURL(configURL string) Option {
	return func(cfg *config) {
		cfg.configURL = configURL
	}
}

// WithAwaitInitialIdentity makes the Track events wait until the initial
// identity is sent.
func WithAwaitInitialIdentity() Option {
	return func(cfg *config) {
		cfg.awaitInitialIdentity = true
	}
}

// WithConfigureCallback sets the callback to be called when the client is
// reconfigured.
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

// withClient sets the default client identification.
func withClient(clientID string, clientType, clientVersion string) Option {
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
