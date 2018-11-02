package version

import (
	"io/ioutil"
	"strings"
)

const (
	collectorVersionFile = "COLLECTOR_VERSION.txt" // check https://hub.docker.com/r/stackrox/collector/tags/

	versionFile = "VERSION.txt"
)

var (
	version          string
	collectorVersion string
)

func init() {
	version, _ = readVersion(versionFile)
	collectorVersion, _ = readVersion(collectorVersionFile)
}

// GetVersion returns the tag of Prevent
func GetVersion() (string, error) {
	if version == "" {
		panic("version string is empty")
	}
	return version, nil
}

// GetCollectorVersion returns the current collector tag
func GetCollectorVersion() string {
	if collectorVersion == "" {
		panic("collector version string is empty")
	}
	return collectorVersion
}

func readVersion(filename string) (string, error) {
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}
