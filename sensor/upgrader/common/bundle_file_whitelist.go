package common

import (
	"path"

	"github.com/stackrox/rox/pkg/set"
)

var (
	// bundleFileWhitelist is a list of files and directories that are part of the sensor bundle but do not need to be
	// considered by the upgrader.
	bundleFileWhitelist = set.NewFrozenStringSet(
		"delete-sensor.sh",
		"docker-auth.sh",
		"sensor.sh",
		"additional-cas/",
		"ca.pem",
		"sensor-key.pem",
		"sensor-cert.pem",
		"collector-key.pem",
		"collector-cert.pem",
		"admission-control-cert.pem",
		"admission-control-key.pem",
	)
)

// IsWhitelistedBundleFile checks if the given file is a whitelisted file, i.e., does not need to be considered by
// the upgrader.
func IsWhitelistedBundleFile(file string) bool {
	if bundleFileWhitelist.Contains(file) {
		return true
	}

	dir := path.Dir(file)
	for dir != "" && dir != "." {
		if bundleFileWhitelist.Contains(dir + "/") {
			return true
		}
		dir = path.Dir(dir)
	}
	return false
}
