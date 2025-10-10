package cvssv2

import (
	"fmt"

	"github.com/facebookincubator/nvdtools/cvss2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/mathutil"
)

var attackVectorMap = map[string]storage.CVSSV2_AttackVector{
	"L": storage.CVSSV2_ATTACK_LOCAL,
	"A": storage.CVSSV2_ATTACK_ADJACENT,
	"N": storage.CVSSV2_ATTACK_NETWORK,
}

var accessComplexityMap = map[string]storage.CVSSV2_AccessComplexity{
	"H": storage.CVSSV2_ACCESS_HIGH,
	"M": storage.CVSSV2_ACCESS_MEDIUM,
	"L": storage.CVSSV2_ACCESS_LOW,
}

var authenticationMap = map[string]storage.CVSSV2_Authentication{
	"M": storage.CVSSV2_AUTH_MULTIPLE,
	"S": storage.CVSSV2_AUTH_SINGLE,
	"N": storage.CVSSV2_AUTH_NONE,
}

var impactMap = map[string]storage.CVSSV2_Impact{
	"N": storage.CVSSV2_IMPACT_NONE,
	"P": storage.CVSSV2_IMPACT_PARTIAL,
	"C": storage.CVSSV2_IMPACT_COMPLETE,
}

var severityMap = map[string]storage.CVSSV2_Severity{
	"U": storage.CVSSV2_UNKNOWN,
	"L": storage.CVSSV2_LOW,
	"M": storage.CVSSV2_MEDIUM,
	"H": storage.CVSSV2_HIGH,
}

// GetSeverityMapProtoVal returns the proto enum value of severity
func GetSeverityMapProtoVal(s string) (storage.CVSSV2_Severity, error) {
	v, ok := severityMap[s]
	if !ok {
		return -1, fmt.Errorf("key %q not found in severityMap", s)
	}
	return v, nil
}

// ParseCVSSV2 parses the vector string and returns an internal representation of CVSS V2
func ParseCVSSV2(vectorStr string) (*storage.CVSSV2, error) {
	vec, err := cvss2.VectorFromString(vectorStr)
	if err != nil {
		return nil, fmt.Errorf("invalid CVSSv2 vector %q: %w", vectorStr, err)
	}
	if err := vec.Validate(); err != nil {
		return nil, fmt.Errorf("invalid CVSSv2 vector %q: %w", vectorStr, err)
	}

	// We only care about base metrics at this time.
	metrics := vec.BaseMetrics

	attackVector := attackVectorMap[metrics.AccessVector.String()]
	accessComplexity := accessComplexityMap[metrics.AccessComplexity.String()]
	authentication := authenticationMap[metrics.Authentication.String()]
	confidentiality := impactMap[metrics.ConfidentialityImpact.String()]
	integrity := impactMap[metrics.IntegrityImpact.String()]
	availability := impactMap[metrics.AvailabilityImpact.String()]

	return storage.CVSSV2_builder{
		Vector:           &vectorStr,
		AttackVector:     &attackVector,
		AccessComplexity: &accessComplexity,
		Authentication:   &authentication,
		Confidentiality:  &confidentiality,
		Integrity:        &integrity,
		Availability:     &availability,
	}.Build(), nil
}

// Severity returns the severity for the cvss v2 score
func Severity(score float32) storage.CVSSV2_Severity {
	switch {
	case score <= 3.9:
		return storage.CVSSV2_LOW
	case score <= 6.9:
		return storage.CVSSV2_MEDIUM
	case score <= 10.0:
		return storage.CVSSV2_HIGH
	}
	return storage.CVSSV2_UNKNOWN
}

// CalculateScores calculates and sets CVSS scores based on the current vector string.
func CalculateScores(cvssV2 *storage.CVSSV2) error {
	vec, err := cvss2.VectorFromString(cvssV2.GetVector())
	if err != nil {
		return fmt.Errorf("parsing: %w", err)
	}
	if err := vec.Validate(); err != nil {
		return fmt.Errorf("validating: %w", err)
	}
	cvssV2.SetScore(float32(vec.BaseScore()))
	cvssV2.SetExploitabilityScore(float32(mathutil.RoundToDecimal(vec.ExploitabilityScore(), 1)))
	cvssV2.SetImpactScore(float32(mathutil.RoundToDecimal(vec.ImpactScore(false), 1)))
	return nil
}
