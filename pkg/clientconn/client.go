package clientconn

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

var (
	logger = logging.LoggerForModule()
)

// GRPCConnection returns a grpc.ClientConn object.
func GRPCConnection(endpoint string) (conn *grpc.ClientConn, err error) {
	return AuthenticatedGRPCConnection(endpoint)
}

// The following is ugly, but, unfortunately, necessary.
// The default GoLang TLS config uses the ServerName field for two things:
// Server Name Indication, and verification of the common name in the client certificate.
// For us, the common name on the Cert must always be the name that we set (example: mtls.CentralCN).
// Most of the time, it is fine to also use this for SNI.
// However, in the specific case of OpenShift routes, it is important that the SNI that we send the
// OpenShift router match the hostname of the route. Otherwise, the router returns a default set
// of certificates, instead of forwarding TLS to Central and giving us Central's certificate.
// There is no way to use a different value for verification versus SNI in the Go library,
// but it _does_ allow you to pass a VerifyPeerCertificate function that gets the rawCerts, and does whatever you want.
// We do that here, explicitly validating the things we want to validate.
// We do NOT set a ServerName directly in the TLS config, so that SNI doesn't break.
// Doing it this way requires setting InsecureSkipVerify to true, so we have to do all the verification
// the standard library does in our VerifyPeerCertificate function.
func verifyPeerCertificateFunc(rootCAs *x509.CertPool) func([][]byte, [][]*x509.Certificate) error {
	// The second argument will always be nil since we set InsecureSkipVerify to true.
	// The code here is extracted from crypto/tls/handshake_client.go in the standard library.
	// As of Go 1.10, the code is in the `func doFullHandshake() error` of the `clientHandshakeState` struct.
	// The call chain from the nearest exported function is
	// "crypto/tls".Conn.Handshake -> "crypto/tls".Conn.clientHandshake -> "crypto/tls".clientHandshakeState.handshake
	return func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
		var leafCertificate *x509.Certificate
		nonLeafCertificates := make([]*x509.Certificate, 0, len(rawCerts)-1)

		for i, asn1Data := range rawCerts {
			cert, err := x509.ParseCertificate(asn1Data)
			if err != nil {
				return fmt.Errorf("failed to parse certificate from server: %s", err)
			}
			if i == 0 {
				leafCertificate = cert
			} else {
				nonLeafCertificates = append(nonLeafCertificates, cert)
			}
		}

		intermediates := x509.NewCertPool()
		for _, cert := range nonLeafCertificates {
			intermediates.AddCert(cert)
		}

		_, err := leafCertificate.Verify(x509.VerifyOptions{
			Roots:         rootCAs,
			DNSName:       mtls.CentralCN.String(),
			Intermediates: intermediates,
		})

		if err != nil {
			// This error will be swallowed by gRPC, so log it here for easier debugging.
			logger.Error(err)
			return fmt.Errorf("cert verification failed: %s", err)
		}

		return nil
	}
}

func tlsConfig(clientCert tls.Certificate, rootCAs *x509.CertPool) *tls.Config {
	return &tls.Config{
		Certificates:          []tls.Certificate{clientCert},
		InsecureSkipVerify:    true,
		VerifyPeerCertificate: verifyPeerCertificateFunc(rootCAs),
	}
}

// AuthenticatedGRPCConnection returns a grpc.ClientConn object that uses
// client certificates found on the local file system.
func AuthenticatedGRPCConnection(endpoint string) (conn *grpc.ClientConn, err error) {
	clientCert, err := mtls.LeafCertificateFromFile()
	if err != nil {
		return nil, fmt.Errorf("client credentials: %s", err)
	}
	rootCAs, err := verifier.TrustedCertPool()
	if err != nil {
		return nil, fmt.Errorf("trusted pool: %s", err)
	}

	creds := credentials.NewTLS(tlsConfig(clientCert, rootCAs))
	return grpc.Dial(endpoint, grpc.WithTransportCredentials(creds), keepAliveDialOption())
}

// UnauthenticatedGRPCConnection returns a grpc.ClientConn object that does not use credentials.
// Deprecated: This is only to be used temporarily until Sensors
// issue certificates to their workers.
func UnauthenticatedGRPCConnection(endpoint string) (conn *grpc.ClientConn, err error) {
	tlsConfig := &tls.Config{
		// TODO(cg): Issue credentials and remove this.
		InsecureSkipVerify: true,
	}
	creds := credentials.NewTLS(tlsConfig)
	return grpc.Dial(endpoint, grpc.WithTransportCredentials(creds))
}

// Parameters for keep alive.
func keepAliveDialOption() grpc.DialOption {
	// Since we are holding open a GRPC stream, enable keep alive.
	// Ping every minute of inactivity, and wait 30 seconds. Do this even when no streams are open (though
	// one should always be open with central.)
	params := keepalive.ClientParameters{
		Time:                1 * time.Minute,
		Timeout:             30 * time.Second,
		PermitWithoutStream: true,
	}
	return grpc.WithKeepaliveParams(params)
}
