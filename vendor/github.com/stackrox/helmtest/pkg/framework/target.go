package framework

import (
	"github.com/stackrox/helmtest/internal/schemas"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
)

// Target is the target to run a test against. This must at the minimum include a chart and default release options
// (such as the release name or the namespace). Capabilities is optional and will default to the standard capabilities
// used by Helm in client-only mode.
type Target struct {
	Chart          *chart.Chart
	ReleaseOptions chartutil.ReleaseOptions
	Capabilities   *chartutil.Capabilities

	SchemaRegistry schemas.Registry
}
