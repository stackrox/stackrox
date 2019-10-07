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
