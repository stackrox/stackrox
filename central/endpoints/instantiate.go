package endpoints

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/tlsconfig"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/mtls/verifier"
)

// TLSConfigurerProvider is a slimmed-down subinterface of tlsconfig.Manager.
type TLSConfigurerProvider interface {
	TLSConfigurer(opts tlsconfig.Options) (verifier.TLSConfigurer, error)
}

type tlsConfigurerProviderForValidation struct{}

func (tlsConfigurerProviderForValidation) TLSConfigurer(_ tlsconfig.Options) (verifier.TLSConfigurer, error) {
	return nil, nil
}

// Instantiate obtains the TLSConfigurer reflecting this TLS configuration.
func (c *TLSConfig) Instantiate(tlsMgr TLSConfigurerProvider) (verifier.TLSConfigurer, error) {
	if c.Disable {
		if len(c.ServerCerts) > 0 || c.ClientAuth != nil {
			return nil, errors.New("if TLS is disabled, neither server certs nor client auth config must be specified")
		}
		return nil, nil
	}

	var opts tlsconfig.Options

	serverCerts := c.ServerCerts
	if len(serverCerts) == 0 {
		serverCerts = defaultServerCerts
	}

	for _, serverCert := range serverCerts {
		switch src := strings.ToLower(serverCert); src {
		case "default":
			opts.ServerCerts = append(opts.ServerCerts, tlsconfig.DefaultTLSCertSource)
		case "service":
			opts.ServerCerts = append(opts.ServerCerts, tlsconfig.ServiceCertSource)
		default:
			return nil, errors.Errorf("unknown server certificate setting %q", src)
		}
	}

	clientAuth := c.ClientAuth
	if clientAuth == nil {
		clientAuth = &defaultClientAuthConfig
	}

	var clientCAs []string
	if clientAuth.CertAuthorities != nil {
		clientCAs = *clientAuth.CertAuthorities
	} else {
		clientCAs = defaultClientCertAuthorities
	}

	if clientAuth.Required {
		if len(clientCAs) == 0 {
			return nil, errors.New("client certificate authentication is required, but no client certificate authorities are configured")
		}
		opts.RequireClientCert = true
	}

	for _, clientCA := range clientCAs {
		switch src := strings.ToLower(clientCA); src {
		case "user":
			opts.ClientCAs = append(opts.ClientCAs, tlsconfig.UserCAsSource)
		case "service":
			opts.ClientCAs = append(opts.ClientCAs, tlsconfig.ServiceCASource)
		default:
			return nil, errors.Errorf("unknown client CA source %q", src)
		}
	}

	return tlsMgr.TLSConfigurer(opts)
}

// Instantiate obtains a server endpoint config reflecting this endpoint configuration, and using the given TLS config
// manager for TLS configuration.
func (c *EndpointConfig) Instantiate(tlsMgr TLSConfigurerProvider) (*grpc.EndpointConfig, error) {
	result := grpc.EndpointConfig{
		ListenEndpoint: c.Listen,
		Optional:       c.Optional,
	}

	tlsConfig := c.TLS
	if tlsConfig == nil {
		tlsConfig = &defaultTLSConfig
	}
	tlsConfigurer, err := tlsConfig.Instantiate(tlsMgr)
	if err != nil {
		return nil, errors.Wrap(err, "instantiating TLS configurer")
	}
	result.TLS = tlsConfigurer

	protos := c.Protocols
	if len(protos) == 0 {
		protos = defaultProtocols
	}

	for _, proto := range protos {
		switch p := strings.ToLower(proto); p {
		case "http":
			result.ServeHTTP = true
		case "grpc":
			result.ServeGRPC = true
		default:
			return nil, errors.Errorf("unknown backend protocol %q", p)
		}
	}

	return &result, nil
}

// Validate checks if the given endpoint configuration is valid.
func (c *EndpointConfig) Validate() error {
	_, err := c.Instantiate(tlsConfigurerProviderForValidation{})
	return err
}
