package all

import (
	_ "github.com/stackrox/rox/central/externalbackups/plugins/gcs"           // Register GCS plugin
	_ "github.com/stackrox/rox/central/externalbackups/plugins/s3/compatible" // Register S3compatible plugin
	_ "github.com/stackrox/rox/central/externalbackups/plugins/s3/std"        // Register S3 plugin
)
