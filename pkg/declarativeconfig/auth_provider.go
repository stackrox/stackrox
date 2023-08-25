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

// Group is representation of storage.Group that supports transformation from YAML.
type Group struct {
	AttributeKey   string `yaml:"key,omitempty"`
	AttributeValue string `yaml:"value,omitempty"`
	RoleName       string `yaml:"role,omitempty"`
}

// OIDCConfig contains config values for OIDC auth provider.
type OIDCConfig struct {
	Issuer string `yaml:"issuer,omitempty"`
	// Depending on callback mode, different OAuth 2.0 would be preferred.
	// Possible values are: auto, post, query, fragment.
	CallbackMode string `yaml:"mode,omitempty"`
	ClientID     string `yaml:"clientID,omitempty"`
	ClientSecret string `yaml:"clientSecret,omitempty"`
	// Disables request for "offline_access" scope from OIDC identity provider.
	DisableOfflineAccessScope bool `yaml:"disableOfflineAccessScope,omitempty"`
}

// SAMLConfig contains config values for SAML 2.0 auth provider.
// There are two ways to configure SAML: static and dynamic.
// For dynamic configuration, you only need to specify spIssuer and metadataURL.
// For static configuration, specify spIssuer, cert, ssoURL, idpIssuer, and nameIdFormat.
type SAMLConfig struct {
	SpIssuer    string `yaml:"spIssuer,omitempty"`
	MetadataURL string `yaml:"metadataURL,omitempty"`
	// SAML 2.0 IdP Certificate in PEM format
	Cert         string `yaml:"cert,omitempty"`
	SsoURL       string `yaml:"ssoURL,omitempty"`
	NameIDFormat string `yaml:"nameIdFormat,omitempty"`
	IDPIssuer    string `yaml:"idpIssuer,omitempty"`
}

// IAPConfig contains config values for IAP auth provider.
type IAPConfig struct {
	Audience string `yaml:"audience,omitempty"`
}

// UserpkiConfig contains config values for User Certificates auth provider.
type UserpkiConfig struct {
	// Certificate authorities in PEM format
	CertificateAuthorities string `yaml:"certificateAuthorities,omitempty"`
}

// OpenshiftConfig contains config values for Openshift auth provider.
// The only value "enable" is a flag which can only be set to true.
// If you don't want auth provider to be Openshift auth provider, don't specify "openshift" section.
type OpenshiftConfig struct {
	Enable bool `yaml:"enable,omitempty"`
}

// AuthProvider is representation of storage.AuthProvider that supports transformation from YAML.
// To specify auth provider type, you need to either specify oidc, saml, iap, userpki config or
// set enableOpenshift variable to true.
type AuthProvider struct {
	Name string `yaml:"name,omitempty"`
	// TODO: ROX-14148 if left empty, no default group should be created
	MinimumRoleName string `yaml:"minimumRole,omitempty"`
	// The UIEndpoint should be given without scheme (http:// | https://) but including the port, e.g. localhost:443
	UIEndpoint string `yaml:"uiEndpoint,omitempty"`
	// The ExtraUIEndpoints should be given without scheme (http:// | https://) but including the port, e.g. localhost:443
	ExtraUIEndpoints   []string            `yaml:"extraUIEndpoints,omitempty"`
	Groups             []Group             `yaml:"groups,omitempty"`
	RequiredAttributes []RequiredAttribute `yaml:"requiredAttributes,omitempty"`
	ClaimMappings      []ClaimMapping      `yaml:"claimMappings,omitempty"`
	OIDCConfig         *OIDCConfig         `yaml:"oidc,omitempty"`
	SAMLConfig         *SAMLConfig         `yaml:"saml,omitempty"`
	IAPConfig          *IAPConfig          `yaml:"iap,omitempty"`
	UserpkiConfig      *UserpkiConfig      `yaml:"userpki,omitempty"`
	OpenshiftConfig    *OpenshiftConfig    `yaml:"openshift,omitempty"`
}

// ConfigurationType returns the AuthProviderConfiguration type.
func (a *AuthProvider) ConfigurationType() ConfigurationType {
	return AuthProviderConfiguration
}
