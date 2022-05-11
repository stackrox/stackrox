package utils

import (
	"fmt"
	"runtime"

	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/version"
	"k8s.io/client-go/rest"
)

// SetUserAgent returns a rest.Config configured to set the UserAgent header when sending HTTP requests to K8s API Server.
// Use SetUserAgent only in operator code otherwise it may panic, for more context see defaults.GetImageFlavorFromEnv
func SetUserAgent(config *rest.Config) *rest.Config {
	config.UserAgent = fmt.Sprintf("%s/v%s %s (%s/%s)", "rhacs-operator", version.GetMainVersion(), defaults.GetImageFlavorNameFromEnv(), runtime.GOOS, runtime.GOARCH)
	return config
}
