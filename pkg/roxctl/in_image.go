package roxctl

import (
	"os"
)

// InMainImage returns whether the code is being executed from within the stackrox/main image
// (the env variable is injected in the Dockerfile). This is helpful to differentiate between
// roxctl being invoked in the image vs as a standalone binary.
func InMainImage() bool {
	return os.Getenv("ROX_ROXCTL_IN_MAIN_IMAGE") == "true"
}
