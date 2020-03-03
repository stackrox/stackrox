package endpoints

// ClientAuthConfig configures the client TLS authentication requirements for an endpoint.
type ClientAuthConfig struct {
	Required        bool      `json:"required,omitempty"`
	CertAuthorities *[]string `json:"certAuthorities,omitempty"`
}

// TLSConfig configures the TLS settings for an endpoint.
type TLSConfig struct {
	Disable     bool              `json:"disable,omitempty"`
	ServerCerts []string          `json:"serverCerts"`
	ClientAuth  *ClientAuthConfig `json:"clientAuth,omitempty"`
}

// EndpointConfig configures a single endpoint through which Central's functionality can be exposed.
type EndpointConfig struct {
	Listen    string     `json:"listen"`
	Optional  bool       `json:"optional,omitempty"` // do not exit with a fatal error if we can't listen on this endpoint
	Protocols []string   `json:"protocols,omitempty"`
	TLS       *TLSConfig `json:"tls,omitempty"` // TLS configuration. If unset, assume
}

// Config configures the exposure configuration for Central through various endpoints.
type Config struct {
	DisableDefault bool             `json:"disableDefault,omitempty"` // if true, do not expose default endpoint at :8443
	Endpoints      []EndpointConfig `json:"endpoints,omitempty"`      // additional endpoints to expose
}
