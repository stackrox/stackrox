package osrelease

import (
	"bufio"
	"regexp"
	"strings"

	"github.com/stackrox/scanner/ext/featurens/util"
)

var (
	osPattern      = regexp.MustCompile(`^ID=(.*)`)
	versionPattern = regexp.MustCompile(`^VERSION_ID=(.*)`)
)

// GetOSAndVersionFromOSRelease returns the value of ID= and VERSION_ID= from /etc/os-release formatted data
func GetOSAndVersionFromOSRelease(data []byte) (os, version string) {
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()

		r := osPattern.FindStringSubmatch(line)
		if len(r) == 2 {
			os = strings.Replace(strings.ToLower(r[1]), "\"", "", -1)
		}

		r = versionPattern.FindStringSubmatch(line)
		if len(r) == 2 {
			version = strings.Replace(strings.ToLower(r[1]), "\"", "", -1)
		}
	}
	return util.NormalizeOSName(os), version
}
