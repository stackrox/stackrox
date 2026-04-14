package all

import (
	"github.com/stackrox/rox/central/externalbackups/plugins/gcs"
	s3compatible "github.com/stackrox/rox/central/externalbackups/plugins/s3/compatible"
	s3std "github.com/stackrox/rox/central/externalbackups/plugins/s3/std"
)

// Init registers all external backup plugins.
// Called explicitly from central/app/app.go instead of package init().
func Init() {
	s3std.Register()
	s3compatible.Register()
	gcs.Register()
}
