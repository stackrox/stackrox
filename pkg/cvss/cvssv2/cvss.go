package cvssv2

import (
	"fmt"
	"strings"

	"github.com/stackrox/stackrox/generated/storage"
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
	cvssV2 := &storage.CVSSV2{
		Vector: vectorStr,
	}

	vectors := strings.Split(vectorStr, "/")
	for _, vector := range vectors {
		vals := strings.Split(vector, ":")
		if len(vals) != 2 {
			return nil, fmt.Errorf("invalid format for vector subfield %q", vector)
		}
		k, v := strings.TrimSpace(vals[0]), strings.TrimSpace(vals[1])
		var ok bool
		switch k {
		case "AV":
			cvssV2.AttackVector, ok = attackVectorMap[v]
		case "AC":
			cvssV2.AccessComplexity, ok = accessComplexityMap[v]
		case "Au":
			cvssV2.Authentication, ok = authenticationMap[v]
		case "C":
			cvssV2.Confidentiality, ok = impactMap[v]
		case "I":
			cvssV2.Integrity, ok = impactMap[v]
		case "A":
			cvssV2.Availability, ok = impactMap[v]
		}
		if !ok {
			return nil, fmt.Errorf("invalid field value %q for %q", v, k)
		}
	}
	return cvssV2, nil
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
