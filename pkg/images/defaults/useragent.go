package defaults

import (
	"fmt"
	"os"
	"runtime"

	"github.com/stackrox/rox/pkg/version"
)

// UserAgent returns a default value to set the UserAgent header when sending HTTP requests to K8s API Server.
// Use UserAgent only in central or operator applications otherwise it may panic, for more context see defaults.GetImageFlavorFromEnv
func UserAgent() string {
	return fmt.Sprintf("%s/v%s %s (%s/%s)", os.Args[0], version.GetMainVersion(), GetImageFlavorFromEnv().Name, runtime.GOOS, runtime.GOARCH)
}
