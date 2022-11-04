package convert

import (
	"regexp"

	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"github.com/stackrox/k8s-cves/pkg/validation"
)

var (
	semverPattern = regexp.MustCompile(`(?:v)?([0-9]+\.[0-9]+\.[0-9]+)(?:[-+]?.*)`)
)

// TruncateVersion converts the given version into a semantic version x.y.z.
// Returns empty string ""
func TruncateVersion(v string) (string, error) {
	vs := semverPattern.FindStringSubmatch(v)
	if len(vs) == 2 {
		return vs[1], nil
	}
	return "", errors.Errorf("unsupported version: %s", v)
}

// GetFixedBy gets the fixed-by version for vStr in vuln.
func GetFixedBy(vStr string, vuln *validation.CVESchema) (string, error) {
	v, err := version.NewVersion(vStr)
	if err != nil {
		return "", err
	}

	for _, affected := range vuln.Affected {
		constraint, err := version.NewConstraint(affected.Range)
		if err != nil {
			return "", err
		}
		if constraint.Check(v) {
			return affected.FixedBy, nil
		}
	}

	return "", nil
}
