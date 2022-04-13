package all

import (
	_ "github.com/stackrox/stackrox/central/externalbackups/plugins/gcs" // Register GCS with the plugins
	_ "github.com/stackrox/stackrox/central/externalbackups/plugins/s3"  // Register S3 with the plugins
)
