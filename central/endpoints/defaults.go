package endpoints

const (
	publicAPIEndpoint = ":8443"
)

var (
	defaultProtocols             = []string{"http", "grpc"}
	defaultServerCerts           = []string{"default", "service"}
	defaultClientCertAuthorities = []string{"user", "service"}

	defaultClientAuthConfig = ClientAuthConfig{
		Required:        false,
		CertAuthorities: &defaultClientCertAuthorities,
	}

	defaultTLSConfig = TLSConfig{
		Disable:     false,
		ServerCerts: defaultServerCerts,
		ClientAuth:  &defaultClientAuthConfig,
	}

	plaintextTLSConfig = TLSConfig{
		Disable: true,
	}

	defaultEndpoint = EndpointConfig{
		Listen:    publicAPIEndpoint,
		Optional:  false,
		Protocols: defaultProtocols,
		TLS:       &defaultTLSConfig,
	}

	defaultConfig = Config{
		DisableDefault: false,
	}
)
