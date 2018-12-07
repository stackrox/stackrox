package cvss

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/generated/storage"
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

// ParseCVSSV2 parses the vector string and returns an internal representation of CVSS V2
func ParseCVSSV2(vectorStr string) (*storage.CVSSV2, error) {
	cvssV2 := &storage.CVSSV2{
		Vector: vectorStr,
	}

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
