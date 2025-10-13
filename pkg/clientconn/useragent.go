package clientconn

import (
	"fmt"
	"runtime"

	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/version"
)

var userAgent string

// The following is the list of component names that tune their User-Agent.
const (
	AdmissionController = "Rox Admission Controller"
	Central             = "Rox Central"
	Compliance          = "Rox Compliance"
	ConfigController    = "Config Controller"
	Roxctl              = "roxctl"
	Sensor              = "Rox Sensor"
	Upgrader            = "Rox Upgrader"
)

func init() {
	SetUserAgent("StackRox")
}

// SetUserAgent formats and sets a value to be used in the User-Agent HTTP
// header for the requests, initiated by a process.
// Note: gorpc-go library will append the header value with its version,
// e.g. grpc-go/1.50.1.
func SetUserAgent(agent string) {
	var ci string
	if testutils.IsRunningInCI() {
		ci = " CI"
	}
	userAgent = fmt.Sprintf("%s/%s (%s; %s)%s", agent, version.GetMainVersion(), runtime.GOOS, runtime.GOARCH, ci)
}

// GetUserAgent returns the previously calculated value, which has to be set
// by the process main function via a call to SetUserAgent().
// Default value is the one produced by SetUserAgent("stackrox").
func GetUserAgent() string {
	return userAgent
}
