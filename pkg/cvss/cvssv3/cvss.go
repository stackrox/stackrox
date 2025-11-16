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

type Writer interface {
	GetVector() string

	SetVector(vector string)
	SetExploitabilityScore(score float32)
	SetImpactScore(score float32)
	SetAttackVector(attackVector storage.CVSSV3_AttackVector)
	SetAttackComplexity(attackComplexity storage.CVSSV3_Complexity)
	SetPrivilegesRequired(privilegesRequired storage.CVSSV3_Privileges)
	SetUserInteraction(userInteraction storage.CVSSV3_UserInteraction)
	SetScope(scope storage.CVSSV3_Scope)
	SetConfidentiality(impact storage.CVSSV3_Impact)
	SetIntegrity(impact storage.CVSSV3_Impact)
	SetAvailability(impact storage.CVSSV3_Impact)
	SetScore(score float32)
	SetSeverity(severity storage.CVSSV3_Severity)
}

// ParseCVSSV3 parses the vector string and returns an internal representation of CVSS V3
func ParseCVSSV3(out Writer, vector string) error {
	vec, err := getValidatedVectorFromString(vector)
	if err != nil {
		return err
	}

	// We only care about base metrics at this time.
	metrics := vec.BaseMetrics

	out.SetVector(vector)
	out.SetAttackVector(attackVectorMap[metrics.AttackVector.String()])
	out.SetAttackComplexity(complexityMap[metrics.AttackComplexity.String()])
	out.SetPrivilegesRequired(privilegesMap[metrics.PrivilegesRequired.String()])
	out.SetUserInteraction(userInteractionMap[metrics.UserInteraction.String()])
	out.SetScope(scopeMap[metrics.Scope.String()])
	out.SetConfidentiality(impactMap[metrics.Confidentiality.String()])
	out.SetIntegrity(impactMap[metrics.Integrity.String()])
	out.SetAvailability(impactMap[metrics.Availability.String()])

	return nil
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
func CalculateScores(io Writer) error {
	vec, err := getValidatedVectorFromString(io.GetVector())
	if err != nil {
		return err
	}
	io.SetScore(float32(vec.BaseScore()))
	io.SetExploitabilityScore(float32(mathutil.RoundToDecimal(vec.ExploitabilityScore(), 1)))
	io.SetImpactScore(float32(mathutil.RoundToDecimal(vec.ImpactScore(), 1)))
	return nil
}

func getValidatedVectorFromString(vector string) (cvss3.Vector, error) {
	vec, err := cvss3.VectorFromString(vector)
	if err != nil {
		return cvss3.Vector{}, fmt.Errorf("invalid CVSSv3 vector %q: %w", vector, err)
	}
	if err := vec.Validate(); err != nil {
		return cvss3.Vector{}, fmt.Errorf("invalid CVSSv3 vector %q: %w", vector, err)
	}

	return vec, nil
}
