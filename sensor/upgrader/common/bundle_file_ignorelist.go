package common

import (
	"path"

	"github.com/stackrox/rox/pkg/set"
)

var (
	// bundleFileIgnorelist is a list of files and directories that are part of the sensor bundle but do not need to be
	// considered by the upgrader.
	bundleFileIgnorelist = set.NewFrozenStringSet(
		"delete-sensor.sh",
		"docker-auth.sh",
		"sensor.sh",
		"additional-ca-sensor.yaml",
		"additional-cas/",
		"ca.pem",
		"sensor-key.pem",
		"sensor-cert.pem",
		"collector-key.pem",
		"collector-cert.pem",
		"admission-control-cert.pem",
		"admission-control-key.pem",
		"ca-setup-sensor.sh",
		"delete-ca-sensor.sh",
		"NOTES.txt",
	)
)

// IsIgnorelistedBundleFile checks if the given file is a baselined file, i.e., does not need to be considered by
// the upgrader.
func IsIgnorelistedBundleFile(file string) bool {
	if bundleFileIgnorelist.Contains(file) {
		return true
	}

	dir := path.Dir(file)
	for dir != "" && dir != "." {
		if bundleFileIgnorelist.Contains(dir + "/") {
			return true
		}
		dir = path.Dir(dir)
	}
	return false
}
