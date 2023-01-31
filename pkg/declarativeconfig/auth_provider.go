package declarativeconfig

// RequiredAttribute is representation of storage.AuthProvider_RequiredAttribute that supports transformation from YAML.
type RequiredAttribute struct {
	AttributeKey   string `yaml:"key,omitempty"`
	AttributeValue string `yaml:"value,omitempty"`
}

// ClaimMapping represents a single entry in "claim_mappings" field in auth provider proto.
type ClaimMapping struct {
	Path string `yaml:"path,omitempty"`
	Name string `yaml:"name,omitempty"`
}

// OIDCConfig contains config values for OIDC auth provider.
type OIDCConfig struct {
	Issuer                    string `yaml:"issuer,omitempty"`
	CallbackMode              string `yaml:"mode,omitempty"`
	ClientID                  string `yaml:"clientID,omitempty"`
	ClientSecret              string `yaml:"clientSecret,omitempty"`
	DisableOfflineAccessScope bool   `yaml:"disableOfflineAccessScope,omitempty"`
	DontUseClientSecretConfig bool   `yaml:"dontUseClientSecretConfig,omitempty"`
}

// SAMLConfig contains config values for SAML auth provider.
type SAMLConfig struct {
	SpIssuer     string `yaml:"spIssuer,omitempty"`
	MetadataURL  string `yaml:"metadataURL,omitempty"`
	CertPEM      string `yaml:"certPEM,omitempty"`
	SsoURL       string `yaml:"ssoURL,omitempty"`
	NameidFormat string `yaml:"nameIdFormat,omitempty"`
}

// IAPConfig contains config values for IAP auth provider.
type IAPConfig struct {
	Audience string `yaml:"audience,omitempty"`
}

// UserpkiConfig contains config values for User Certificates auth provider.
type UserpkiConfig struct {
	Keys string `yaml:"keys,omitempty"`
}

// AuthProvider is representation of storage.AuthProvider that supports transformation from YAML.
// To specify auth provider type, you need to either specify oidc, saml, iap, userpki config or
// set enableOpenshift variable to true.
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
	IAPConfig          *IAPConfig          `yaml:"iap,omitempty"`
	UserpkiConfig      *UserpkiConfig      `yaml:"userpki,omitempty"`
	EnableOpenshift    bool                `yaml:"enableOpenshift,omitempty"`
}

// Group is representation of storage.Group that supports transformation from YAML.
type Group struct {
	AttributeKey   string `yaml:"key,omitempty"`
	AttributeValue string `yaml:"value,omitempty"`
	RoleName       string `yaml:"role,omitempty"`
}
