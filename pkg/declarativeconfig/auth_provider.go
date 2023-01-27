package declarativeconfig

type RequiredAttribute struct {
	AttributeKey   string `yaml:"key,omitempty"`
	AttributeValue string `yaml:"value,omitempty"`
}

type ClaimMapping struct {
	PathToClaim    string `yaml:"pathToClaim,omitempty"`
	MapToClaimName string `yaml:"mapTo,omitempty"`
}

type OIDCConfig struct {
	Issuer                    string `yaml:"issuer,omitempty"`
	CallbackMode              string `yaml:"mode,omitempty"`
	ClientID                  string `yaml:"clientID,omitempty"`
	ClientSecret              string `yaml:"clientSecret,omitempty"`
	DisableOfflineAccessScope bool   `yaml:"disableOfflineAccessScope,omitempty"`
	DontUseClientSecretConfig bool   `yaml:"dontUseClientSecretConfig,omitempty"`
}

type SAMLConfig struct {
	// TODO: add more fields
	// "sp_issuer"
	// "idp_metadata_url"
	// "idp_cert_pem"
	// "idp_sso_url"
	// "idp_nameid_format"
	SPIssuer string `yaml:"spIssuer,omitempty"`
}

// AuthProvider is representation of storage.PermissionSet that supports transformation from YAML.
type AuthProvider struct {
	Name               string              `yaml:"name,omitempty"`
	MinimumRoleName    string              `yaml:"minimumRole,omitempty"`
	UIEndpoint         string              `yaml:"uiEndpoint,omitempty"`
	ExtraUIEndpoints   []string            `yaml:"extraUIEndpoints,omitempty"`
	Groups             []Group             `yaml:"groups,omitempty"`
	RequiredAttributes []RequiredAttribute `yaml:"requiredAttributes,omitempty"`
	ClaimMappings      []ClaimMapping      `yaml:"claimMappings,omitempty"`
	OIDCConfig         *OIDCConfig         `yaml:"oidc,omitempty"`
	SAMLConfig         *SAMLConfig         `yaml:"saml,omitempty"`
}

type Group struct {
	AttributeKey   string `yaml:"key,omitempty"`
	AttributeValue string `yaml:"value,omitempty"`
	RoleName       string `yaml:"role,omitempty"`
}
