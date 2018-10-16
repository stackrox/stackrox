package version

import (
	"io/ioutil"
	"os"
	"strings"
	"sync"
)

const (
	collectorTagEnvVar  = "ROX_COLLECTOR_TAG"
	defaultCollectorTag = "1.6.0-42-gb4b63aee" // check https://hub.docker.com/r/stackrox/collector/tags/

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
	collectorTag := os.Getenv(collectorTagEnvVar)
	if collectorTag == "" {
		collectorTag = defaultCollectorTag
	}
	return collectorTag
}
