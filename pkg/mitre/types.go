package mitre

// Various values for different mitre fields.
const (
	mitreAttackDataSrc string = "mitre-attack"

	metadata      mitreObjectType = "x-mitre-collection"
	attackPattern mitreObjectType = "attack-pattern"
	tactic        mitreObjectType = "x-mitre-tactic"

	Enterprise Domain = "enterprise-attack"
	Mobile     Domain = "mobile-attack"

	Android         Platform = "Android"
	AzureAD         Platform = "Azure AD"
	Container       Platform = "Containers"
	GoogleWorkspace Platform = "Google Workspace"
	Iaas            Platform = "IaaS"
	Linux           Platform = "Linux"
	MacOS           Platform = "macOS"
	Network         Platform = "Network"
	Office365       Platform = "Office 365"
	PRE             Platform = "PRE"
	Saas            Platform = "SaaS"
	Windows         Platform = "Windows"
)

type mitreObjectType string

// Domain is a wrapper around `x_mitre_domains` field values in MITRE ATT&CK JSON. It represents the top level MITRE ATT&CK matrix.
type Domain string

func (d Domain) String() string {
	return string(d)
}

// Platform is a wrapper around `x_mitre_platforms` field values in MITRE ATT&CK JSON. represents the MITRE ATT&CK platform(/matrix).
type Platform string

func (p Platform) String() string {
	return string(p)
}

type mitreBundle struct {
	Objects []mitreObject `json:"objects"`
}

type mitreObject struct {
	Name                 string              `json:"name"`
	Description          string              `json:"description"`
	Type                 mitreObjectType     `json:"type"`
	ExternalReferences   []externalReference `json:"external_references"`
	XMitreShortname      string              `json:"x_mitre_shortname"`
	XMitreIsSubtechnique bool                `json:"x_mitre_is_subtechnique"`
	XMitreDomains        []Domain            `json:"x_mitre_domains"`
	XMitrePlatforms      []Platform          `json:"x_mitre_platforms"`
	KillChainPhases      []killChainPhase    `json:"kill_chain_phases"`
	Version              string              `json:"x_mitre_version"`
}

type externalReference struct {
	ExternalID string `json:"external_id"`
	SourceName string `json:"source_name"`
}

type killChainPhase struct {
	KillChainName string `json:"kill_chain_name"`
	PhaseName     string `json:"phase_name"`
}
