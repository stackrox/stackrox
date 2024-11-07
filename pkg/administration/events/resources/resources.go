package resources

import "github.com/stackrox/rox/pkg/sac/resources"

// Resources used in administration events.
const (
	APIToken    = "API Token"
	Backup      = "Backup"
	CloudSource = "Cloud Source"
	Notifier    = "Notifier"
)

// Resources used in administration events.
var (
	Cluster = resources.Cluster.String()
	Image   = resources.Image.String()
	Node    = resources.Node.String()
)
