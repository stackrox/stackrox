package common

import (
	"fmt"
	"os"
	"runtime"
	"strconv"

	"github.com/stackrox/rox/pkg/version"
)

var userAgent string

// GetUserAgent returns a value for the User-Agent HTTP header.
func GetUserAgent() string {
	if userAgent != "" {
		return userAgent
	}
	var ci string
	if v, ok := os.LookupEnv("CI"); ok {
		if v == "" {
			ci = " CI"
		} else if value, err := strconv.ParseBool(v); err == nil && value {
			ci = " CI"
		}
	}
	userAgent = fmt.Sprintf("roxctl/%s (%s; %s)%s", version.GetMainVersion(), runtime.GOOS, runtime.GOARCH, ci)
	return userAgent
}
