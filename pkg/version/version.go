package version

import (
	"io/ioutil"
	"strings"
	"sync"
)

const (
	collectorTag = "1.6.0-18-g34ae6465" // check https://hub.docker.com/r/stackrox/collector/tags/

	versionFile = "VERSION"
)

var (
	version string
	err     error
	once    sync.Once
)

// GetVersion returns the tag of Prevent
func GetVersion() (string, error) {
	once.Do(func() {
		var versionBytes []byte
		versionBytes, err = ioutil.ReadFile(versionFile)
		if err != nil {
			return
		}
		version = strings.TrimSpace(string(versionBytes))
	})
	return version, err
}

// GetCollectorVersion returns the current collector tag
func GetCollectorVersion() string {
	return collectorTag
}
