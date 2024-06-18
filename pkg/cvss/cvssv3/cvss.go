package cvssv3

import (
	"fmt"

	"github.com/facebookincubator/nvdtools/cvss3"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/mathutil"
)

var attackVectorMap = map[string]storage.CVSSV3_AttackVector{
	"L": storage.CVSSV3_ATTACK_LOCAL,
	"A": storage.CVSSV3_ATTACK_ADJACENT,
	"N": storage.CVSSV3_ATTACK_NETWORK,
	"P": storage.CVSSV3_ATTACK_PHYSICAL,
}

var complexityMap = map[string]storage.CVSSV3_Complexity{
	"H": storage.CVSSV3_COMPLEXITY_HIGH,
	"L": storage.CVSSV3_COMPLEXITY_LOW,
}

var impactMap = map[string]storage.CVSSV3_Impact{
	"N": storage.CVSSV3_IMPACT_NONE,
	"L": storage.CVSSV3_IMPACT_LOW,
	"H": storage.CVSSV3_IMPACT_HIGH,
}

var privilegesMap = map[string]storage.CVSSV3_Privileges{
	"N": storage.CVSSV3_PRIVILEGE_NONE,
	"L": storage.CVSSV3_PRIVILEGE_LOW,
	"H": storage.CVSSV3_PRIVILEGE_HIGH,
}

var userInteractionMap = map[string]storage.CVSSV3_UserInteraction{
	"N": storage.CVSSV3_UI_NONE,
	"R": storage.CVSSV3_UI_REQUIRED,
}

var scopeMap = map[string]storage.CVSSV3_Scope{
	"U": storage.CVSSV3_UNCHANGED,
	"C": storage.CVSSV3_CHANGED,
}

var severityMap = map[string]storage.CVSSV3_Severity{
	"U": storage.CVSSV3_UNKNOWN,
	"N": storage.CVSSV3_NONE,
	"L": storage.CVSSV3_LOW,
	"M": storage.CVSSV3_MEDIUM,
	"H": storage.CVSSV3_HIGH,
	"C": storage.CVSSV3_CRITICAL,
}

// GetSeverityMapProtoVal returns the proto enum value of severity
func GetSeverityMapProtoVal(s string) (storage.CVSSV3_Severity, error) {
	v, ok := severityMap[s]
	if !ok {
		return -1, fmt.Errorf("key %q not found in severityMap", s)
	}
	return v, nil
}

// ParseCVSSV3 parses the vector string and returns an internal representation of CVSS V3
func ParseCVSSV3(vectorStr string) (*storage.CVSSV3, error) {
	vec, err := cvss3.VectorFromString(vectorStr)
	if err != nil {
		return nil, fmt.Errorf("invalid CVSSv3 vector %q: %w", vectorStr, err)
	}
	if err := vec.Validate(); err != nil {
		return nil, fmt.Errorf("invalid CVSSv3 vector %q: %w", vectorStr, err)
	}

	// We only care about base metrics at this time.
	metrics := vec.BaseMetrics

	return &storage.CVSSV3{
		Vector:             vectorStr,
		AttackVector:       attackVectorMap[metrics.AttackVector.String()],
		AttackComplexity:   complexityMap[metrics.AttackComplexity.String()],
		PrivilegesRequired: privilegesMap[metrics.PrivilegesRequired.String()],
		UserInteraction:    userInteractionMap[metrics.UserInteraction.String()],
		Scope:              scopeMap[metrics.Scope.String()],
		Confidentiality:    impactMap[metrics.Confidentiality.String()],
		Integrity:          impactMap[metrics.Integrity.String()],
		Availability:       impactMap[metrics.Availability.String()],
	}, nil
}

// Severity returns the severity for the cvss v3 score
func Severity(score float32) storage.CVSSV3_Severity {
	switch {
	case score == 0.0:
		return storage.CVSSV3_NONE
	case score <= 3.9:
		return storage.CVSSV3_LOW
	case score <= 6.9:
		return storage.CVSSV3_MEDIUM
	case score <= 8.9:
		return storage.CVSSV3_HIGH
	case score <= 10.0:
		return storage.CVSSV3_CRITICAL
	}
	return storage.CVSSV3_UNKNOWN
}

// CalculateScores calculates and sets CVSS scores based on the current vector string.
func CalculateScores(cvssV3 *storage.CVSSV3) error {
	vec, err := cvss3.VectorFromString(cvssV3.GetVector())
	if err != nil {
		return fmt.Errorf("parsing: %w", err)
	}
	if err := vec.Validate(); err != nil {
		return fmt.Errorf("validating: %w", err)
	}
	cvssV3.Score = float32(vec.BaseScore())
	cvssV3.ExploitabilityScore = float32(mathutil.RoundToDecimal(vec.ExploitabilityScore(), 1))
	cvssV3.ImpactScore = float32(mathutil.RoundToDecimal(vec.ImpactScore(), 1))
	return nil
}
