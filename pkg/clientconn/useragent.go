package clientconn

import (
	"fmt"
	"os"
	"runtime"
	"strconv"

	"github.com/stackrox/rox/pkg/version"
)

var userAgent string

func init() {
	SetUserAgent("stackrox")
}

// SetUserAgent formats and sets a value for the User-Agent for the process.
func SetUserAgent(agent string) {
	var ci string
	if v, ok := os.LookupEnv("CI"); ok {
		if v == "" {
			ci = " CI"
		} else if value, err := strconv.ParseBool(v); err == nil && value {
			ci = " CI"
		}
	}
	userAgent = fmt.Sprintf("%s/%s (%s; %s)%s", agent, version.GetMainVersion(), runtime.GOOS, runtime.GOARCH, ci)
}

// GetUserAgent returns the previously defined value
// for the User-Agent HTTP header.
func GetUserAgent() string {
	return userAgent
}
