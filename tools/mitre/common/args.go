package common

import "github.com/stackrox/stackrox/pkg/mitre"

const (
	// MitreEnterpriseAttackSrcURL is the location where the most recent MIRE Enterprise ATT&CK data is available.
	MitreEnterpriseAttackSrcURL = "https://raw.githubusercontent.com/mitre-attack/attack-stix-data/master/enterprise-attack/enterprise-attack.json"

	// MitreMobileAttackSrcURL is the location where the most recent MIRE Mobile ATT&CK data is available.
	MitreMobileAttackSrcURL = "https://raw.githubusercontent.com/mitre-attack/attack-stix-data/master/mobile-attack/mobile-attack.json"

	// DefaultOutFile is the default filename to which MITRE ATT&CK is written.
	DefaultOutFile = "mitre.json"
)

// Following represent the various MITRE ATT&CK domains and platforms. Extracted from v9 release.
var (
	MitreDomainsCmdArgs   []string
	MitrePlatformsCmdArgs []string

	CmdArgMitreDomainMap = map[string]*domainWrapper{
		"enterprise": {
			URL:    MitreEnterpriseAttackSrcURL,
			Domain: mitre.Enterprise,
		},
		"mobile": {
			URL:    MitreMobileAttackSrcURL,
			Domain: mitre.Mobile,
		},
	}

	CmdArgMitrePlatformMap = map[string]mitre.Platform{
		"android":   mitre.Android,
		"azureAD":   mitre.AzureAD,
		"container": mitre.Container,
		"gsuite":    mitre.GoogleWorkspace,
		"iaas":      mitre.Iaas,
		"linux":     mitre.Linux,
		"macos":     mitre.MacOS,
		"network":   mitre.Network,
		"office365": mitre.Office365,
		"pre":       mitre.PRE,
		"windows":   mitre.Windows,
		"saas":      mitre.Saas,
	}
)

type domainWrapper struct {
	URL    string
	Domain mitre.Domain
}

func init() {
	for k := range CmdArgMitreDomainMap {
		MitreDomainsCmdArgs = append(MitreDomainsCmdArgs, k)
	}

	for k := range CmdArgMitrePlatformMap {
		MitrePlatformsCmdArgs = append(MitrePlatformsCmdArgs, k)
	}
}
