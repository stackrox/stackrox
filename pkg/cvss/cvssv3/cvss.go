package cvssv3

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/generated/storage"
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
	cvssV3 := &storage.CVSSV3{
		Vector: vectorStr,
	}
	vectorStr = strings.TrimPrefix(vectorStr, "CVSS:3.0/")
	vectorStr = strings.TrimPrefix(vectorStr, "CVSS:3.1/")

	vectors := strings.Split(vectorStr, "/")
	for _, vector := range vectors {
		vals := strings.Split(vector, ":")
		if len(vals) != 2 {
			return nil, fmt.Errorf("Invalid format for vector subfield %q", vector)
		}
		k, v := strings.TrimSpace(vals[0]), strings.TrimSpace(vals[1])
		var ok bool
		switch k {
		case "AV":
			cvssV3.AttackVector, ok = attackVectorMap[v]
		case "AC":
			cvssV3.AttackComplexity, ok = complexityMap[v]
		case "PR":
			cvssV3.PrivilegesRequired, ok = privilegesMap[v]
		case "UI":
			cvssV3.UserInteraction, ok = userInteractionMap[v]
		case "S":
			cvssV3.Scope, ok = scopeMap[v]
		case "C":
			cvssV3.Confidentiality, ok = impactMap[v]
		case "I":
			cvssV3.Integrity, ok = impactMap[v]
		case "A":
			cvssV3.Availability, ok = impactMap[v]
		}
		if !ok {
			return nil, fmt.Errorf("invalid field value %q for %q", v, k)
		}
	}
	return cvssV3, nil
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
