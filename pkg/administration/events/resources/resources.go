package resources

import "github.com/stackrox/rox/pkg/sac/resources"

// Resources used in administration events.
const (
	APIToken = "API Token"
	Notifier = "Notifier"
)

// Resources used in administration events.
var (
	Image   = resources.Image.String()
	Cluster = resources.Cluster.String()
	Node    = resources.Node.String()
)
