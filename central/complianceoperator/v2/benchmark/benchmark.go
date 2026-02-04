package benchmark

import (
	"fmt"
	"regexp"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

type benchmarkInfo struct {
	shortName string
	provider  string
}

var (
	compiledRegexMap = map[*regexp.Regexp]benchmarkInfo{
		regexp.MustCompile(".+-stig$|.+-stig-.+"):         {shortName: "STIG", provider: "DISA"},
		regexp.MustCompile(".+-bsi$|.+-bsi-.+"):           {shortName: "BSI", provider: "Germany’s Federal Office for Information Security"},
		regexp.MustCompile(".+-e8$|.+-e8-.+"):             {shortName: "E8", provider: "ACSC"},
		regexp.MustCompile(".+-nerc-cip$|.+-nerc-cip-.+"): {shortName: "NERC-CIP", provider: "NERC"},
		regexp.MustCompile(".+-pci-dss$|.+-pci-dss-.+"):   {shortName: "PCI-DSS", provider: "PCI"},

		regexp.MustCompile("^ocp4-cis$|^ocp4-cis-.+"):               {shortName: "CIS-OCP", provider: "CIS"},
		regexp.MustCompile("^ocp4-high$|^ocp4-high-.+"):             {shortName: "NIST-800-53", provider: "NIST"},
		regexp.MustCompile("^ocp4-moderate$|^ocp4-moderate-.+"):     {shortName: "NIST-800-53", provider: "NIST"},
		regexp.MustCompile("^rhcos4-high$|^rhcos4-high-.+"):         {shortName: "NIST-800-53", provider: "NIST"},
		regexp.MustCompile("^rhcos4-moderate$|^rhcos4-moderate-.+"): {shortName: "NIST-800-53", provider: "NIST"},
	}

	log = logging.LoggerForModule()
)

func getBenchmarkInfoForProfileName(profileName string) (*benchmarkInfo, error) {
	for compiledRegex, info := range compiledRegexMap {
		if compiledRegex.MatchString(profileName) {
			return &info, nil
		}
	}

	return nil, fmt.Errorf("could not find benchmark info for profile %s", profileName)
}

// GetBenchmarkShortNameFromProfileName returns benchmark short name from profile name.
func GetBenchmarkShortNameFromProfileName(profileName string) string {
	info, err := getBenchmarkInfoForProfileName(profileName)
	if err != nil {
		log.Warn(errors.Wrap(err, "get benchmark by profile name"))

		return ""
	}

	return info.shortName
}

// GetBenchmarkFromProfile returns the benchmarks for the given profile name
func GetBenchmarkFromProfile(profile *storage.ComplianceOperatorProfileV2) (*storage.ComplianceOperatorBenchmarkV2, error) {
	info, err := getBenchmarkInfoForProfileName(profile.GetName())
	if err != nil {
		log.Warn(errors.Wrap(err, "get benchmark by profile"))

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
