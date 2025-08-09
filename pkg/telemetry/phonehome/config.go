package phonehome

import (
	"time"

	"github.com/stackrox/rox/pkg/eventual"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// TenantIDLabel is the name of the k8s object label that holds the cloud
// services tenant ID. The value of the label becomes the group ID if not empty.
const TenantIDLabel = "rhacs.redhat.com/tenant"

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

	// StorageKey should be eventually set by the client.
	// Any attempt to send telemetry data will wait for the key.
	StorageKey   *eventual.Value[string]
	Endpoint     string
	PushInterval time.Duration
	BatchSize    int

	// The period of identity gathering. Default is 1 hour.
	GatherPeriod time.Duration

	// ConfigURL is the URL from where to download runtime configuration.
	ConfigURL string
	// OnReconfigure is called every time a remote configuration is downloaded.
	OnReconfigure func(*RuntimeConfig)

	// AwaitInitialIdentity tells whether Track events are blocked until
	// InitialIdentitySent() is called on the client.
	AwaitInitialIdentity bool
}
