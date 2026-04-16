package probeupload

import "regexp"

var (
	moduleNameRegexStr = `(?:[0-9a-f]{64}|\d+\.\d+\.\d+(?:-rc\d+)?)`
	moduleNameRegex    = regexp.MustCompile(`^` + moduleNameRegexStr + `$`)

	probeNameRegexStr = `collector-(?:ebpf-\d+\.[^/]+\.o|\d+\.[^/]+\.ko)\.gz`
	probeNameRegex    = regexp.MustCompile(`^` + probeNameRegexStr + `$`)

	moduleAndProbeNameRegex = regexp.MustCompile(`^` + moduleNameRegexStr + `/` + probeNameRegexStr + `$`)
)

// IsValidModuleVersion returns whether str is a valid module version.
func IsValidModuleVersion(str string) bool {
	return moduleNameRegex.MatchString(str)
}

// IsValidProbeName returns whether str is a valid file name (basename) for a probe.
func IsValidProbeName(str string) bool {
	return probeNameRegex.MatchString(str)
}

// IsValidFilePath returns whether str is a valid file path for a probe.
func IsValidFilePath(str string) bool {
	return moduleAndProbeNameRegex.MatchString(str)
}
