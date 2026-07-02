package manual

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// VulnerabilityOverride represents a manual override for vulnerability metadata.
type VulnerabilityOverride struct {
	Name               string `yaml:"Name"`
	Description        string `yaml:"Description,omitempty"`
	Severity           string `yaml:"Severity,omitempty"`
	NormalizedSeverity string `yaml:"NormalizedSeverity,omitempty"`
	FixedInVersion     string `yaml:"FixedInVersion,omitempty"`
	Links              string `yaml:"Links,omitempty"`
}

// overridesDocument is the wrapper structure for the YAML document.
type overridesDocument struct {
	Vulnerabilities []VulnerabilityOverride `yaml:"vulnerabilities"`
}

// ParseOverrides parses YAML data containing vulnerability overrides.
// Expected format:
//
//	vulnerabilities:
//	  - Name: CVE-2023-9999
//	    Severity: Critical
//	    NormalizedSeverity: Critical
//	    FixedInVersion: "1.0.1"
func ParseOverrides(data []byte) ([]VulnerabilityOverride, error) {
	var doc overridesDocument

	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, errors.Wrap(err, "failed to parse vulnerability overrides YAML")
	}

	return doc.Vulnerabilities, nil
}
