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

type Writer interface {
	GetVector() string

	SetVector(vector string)
	SetAttackVector(attackVector storage.CVSSV2_AttackVector)
	SetAccessComplexity(accessComplexity storage.CVSSV2_AccessComplexity)
	SetAuthentication(authentication storage.CVSSV2_Authentication)
	SetConfidentiality(impact storage.CVSSV2_Impact)
	SetIntegrity(impact storage.CVSSV2_Impact)
	SetAvailability(impact storage.CVSSV2_Impact)
	SetExploitabilityScore(score float32)
	SetImpactScore(score float32)
	SetScore(score float32)
	SetSeverity(severity storage.CVSSV2_Severity)
}

// ParseCVSSV2 parses the vector string and returns an internal representation of CVSS V2
func ParseCVSSV2(out Writer, vector string) error {
	vec, err := getValidatedVectorFromString(vector)
	if err != nil {
		return err
	}

	metrics := vec.BaseMetrics

	out.SetVector(vector)
	out.SetAttackVector(attackVectorMap[metrics.AccessVector.String()])
	out.SetAccessComplexity(accessComplexityMap[metrics.AccessComplexity.String()])
	out.SetAuthentication(authenticationMap[metrics.Authentication.String()])
	out.SetConfidentiality(impactMap[metrics.ConfidentialityImpact.String()])
	out.SetIntegrity(impactMap[metrics.IntegrityImpact.String()])
	out.SetAvailability(impactMap[metrics.AvailabilityImpact.String()])

	return nil
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
func CalculateScores(io Writer) error {
	vec, err := getValidatedVectorFromString(io.GetVector())
	if err != nil {
		return err
	}
	io.SetScore(float32(vec.BaseScore()))
	io.SetExploitabilityScore(float32(mathutil.RoundToDecimal(vec.ExploitabilityScore(), 1)))
	io.SetImpactScore(float32(mathutil.RoundToDecimal(vec.ImpactScore(false), 1)))
	return nil
}

func getValidatedVectorFromString(vector string) (cvss2.Vector, error) {
	vec, err := cvss2.VectorFromString(vector)
	if err != nil {
		return cvss2.Vector{}, fmt.Errorf("invalid CVSSv2 vector %q: %w", vector, err)
	}
	if err := vec.Validate(); err != nil {
		return cvss2.Vector{}, fmt.Errorf("invalid CVSSv2 vector %q: %w", vector, err)
	}

	return vec, nil
}
