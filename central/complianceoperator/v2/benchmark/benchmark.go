package benchmark

import (
	"fmt"
	"regexp"

	"github.com/stackrox/rox/generated/storage"
)

type benchmarkInfo struct {
	shortName string
	provider  string
}

var regexMap = map[string]benchmarkInfo{
	".+-stig$|.+-stig-.+":         {shortName: "STIG", provider: "DISA"},
	".+-bsi$|.+-bsi-.+":           {shortName: "BSI", provider: "Germany’s Federal Office for Information Security"},
	".+-e8$|.+-e8-.+":             {shortName: "E8", provider: "ACSC"},
	".+-nerc-cip$|.+-nerc-cip-.+": {shortName: "NERC-CIP", provider: "NERC"},
	".+-pci-dss$|.+-pci-dss-.+":   {shortName: "PCI-DSS", provider: "PCI"},

	"^ocp4-cis$|^ocp4-cis-.+":              {shortName: "CIS-OCP", provider: "CIS"},
	"^ocp4-high$|^ocp4-high-.+":            {shortName: "NIST-800-53", provider: "NIST"},
	"^ocp4-moderate|^ocp4-moderate-.+":     {shortName: "NIST-800-53", provider: "NIST"},
	"^rhcos4-high$|^rhcos4-high-.+":        {shortName: "NIST-800-53", provider: "NIST"},
	"^rhcos4-moderate|^rhcos4-moderate-.+": {shortName: "NIST-800-53", provider: "NIST"},
}

func getBenchmarkInfoForProfileName(profileName string) (*benchmarkInfo, error) {
	for regexKey, info := range regexMap {
		match, _ := regexp.MatchString(regexKey, profileName)
		if match {
			return &info, nil
		}
	}

	return nil, fmt.Errorf("could not find benchmark info for profile %s", profileName)
}

// GetBenchmarkShortNameFromProfileName returns benchmark short name from profile name.
func GetBenchmarkShortNameFromProfileName(profileName string) string {
	info, err := getBenchmarkInfoForProfileName(profileName)
	if err != nil {
		return ""
	}

	return info.shortName
}

// GetBenchmarkFromProfile returns the benchmarks for the given profile name
func GetBenchmarkFromProfile(profile *storage.ComplianceOperatorProfileV2) (*storage.ComplianceOperatorBenchmarkV2, error) {
	info, err := getBenchmarkInfoForProfileName(profile.GetName())
	if err != nil {
		return &storage.ComplianceOperatorBenchmarkV2{}, nil
	}

	benchmark := &storage.ComplianceOperatorBenchmarkV2{
		Name:      profile.GetTitle(),
		Version:   profile.GetProfileVersion(),
		Provider:  info.provider,
		ShortName: info.shortName,
	}

	return benchmark, nil
}
