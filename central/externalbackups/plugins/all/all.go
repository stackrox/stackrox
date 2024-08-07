package all

import (
	_ "github.com/stackrox/rox/central/externalbackups/plugins/gcs"          // Register GCS plugin
	_ "github.com/stackrox/rox/central/externalbackups/plugins/s3"           // Register S3 plugin
	_ "github.com/stackrox/rox/central/externalbackups/plugins/s3compatible" // Register S3compatible plugin
)
