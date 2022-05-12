package utils

import (
	"fmt"
	"runtime"

	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/version"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
)

// GetConfigOrDie returns the default *rest.Config for the operator's kubernetes client
func GetConfigOrDie() *rest.Config {
	config := ctrl.GetConfigOrDie()
	config.UserAgent = fmt.Sprintf("%s/v%s %s (%s/%s)", "rhacs-operator", version.GetMainVersion(), defaults.GetImageFlavorNameFromEnv(), runtime.GOOS, runtime.GOARCH)
	return config
}
