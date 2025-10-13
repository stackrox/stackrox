package tlsconfig

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

type managerTestSuite struct {
	suite.Suite
}

func TestManager(t *testing.T) {
	suite.Run(t, new(managerTestSuite))
}

func (s *managerTestSuite) SetupSuite() {

	ca, err := certgen.GenerateCA()
	s.Require().NoError(err)

	testCertDir := s.T().TempDir()

	caFile := filepath.Join(testCertDir, "ca.pem")
	s.Require().NoError(os.WriteFile(caFile, ca.CertPEM(), 0644))
	caKeyFile := filepath.Join(testCertDir, "ca-key.pem")
	s.Require().NoError(os.WriteFile(caKeyFile, ca.KeyPEM(), 0600))

	centralCert, err := ca.IssueCertForSubject(mtls.CentralSubject)
	s.Require().NoError(err)

	certFile := filepath.Join(testCertDir, "cert.pem")
	s.Require().NoError(os.WriteFile(certFile, centralCert.CertPEM, 0644))
	keyFile := filepath.Join(testCertDir, "key.pem")
	s.Require().NoError(os.WriteFile(keyFile, centralCert.KeyPEM, 0600))

	s.T().Setenv(mtls.CAFileEnvName, caFile)
	s.T().Setenv(mtls.CAKeyFileEnvName, caKeyFile)
	s.T().Setenv(mtls.CertFilePathEnvName, certFile)
	s.T().Setenv(mtls.KeyFileEnvName, keyFile)
}

func (s *managerTestSuite) TestNoExtraCertIssuedInStackRoxNamespace() {
	mgr, err := newManager(namespaces.StackRox)
	s.Require().NoError(err)

	defaultCert := testutils.IssueSelfSignedCert(s.T(), "my-central.example.org")
	mgr.UpdateDefaultTLSCertificate(&defaultCert)

	s.Len(mgr.internalCerts, 1)
	s.testConnectionWithManager(mgr, []string{"", "central.stackrox", "central.stackrox.svc"}, []string{"not-central.stackrox.svc", "central.alt-ns.svc"})
}

func (s *managerTestSuite) TestExtraCertIssuedInStackRoxNamespace() {
	mgr, err := newManager("alt-ns")
	s.Require().NoError(err)

	defaultCert := testutils.IssueSelfSignedCert(s.T(), "my-central.example.org")
	mgr.UpdateDefaultTLSCertificate(&defaultCert)

	s.Len(mgr.internalCerts, 2)
	s.testConnectionWithManager(mgr, []string{"", "central.stackrox", "central.stackrox.svc", "central.alt-ns", "central.alt-ns.svc"}, []string{"not-central.stackrox.svc", "not-central.alt-ns"})
}

func (s *managerTestSuite) testConnectionWithManager(mgr *managerImpl, acceptedServerNames []string, rejectedServerNames []string) {
	configurer, err := mgr.TLSConfigurer(Options{
		ServerCerts: []ServerCertSource{DefaultTLSCertSource, ServiceCertSource},
	})
	s.Require().NoError(err)

	serverTLSConf, err := configurer.TLSConfig()
	s.Require().NoError(err)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	s.Require().NoError(err)
	server := tls.NewListener(lis, serverTLSConf)

	serverErrC := make(chan error, 1)
	serverCtx, cancelServerCtx := context.WithCancel(context.Background())
	go func() {
		defer cancelServerCtx() // make sure dials/handshakes don't block if the server exits prematurely
		conn, err := server.Accept()
		for ; err == nil; conn, err = server.Accept() {
			_ = conn.(*tls.Conn).HandshakeContext(serverCtx) // client takes care of error checking
			go func(c net.Conn) { _ = c.Close() }(conn)      // Close might block due to a grace period
		}
		serverErrC <- err
	}()

	for _, serverName := range acceptedServerNames {
		clientTLSConf, err := clientconn.TLSConfig(mtls.CentralSubject, clientconn.TLSConfigOptions{
			ServerName: serverName,
			RootCAs:    getCertPool(mgr.internalTrustRoots),
		})
		if !s.NoError(err) {
			continue
		}
		conn, err := (&net.Dialer{}).DialContext(serverCtx, lis.Addr().Network(), lis.Addr().String())
		if !s.NoError(err) {
			continue
		}
		tlsConn := tls.Client(conn, clientTLSConf)
		s.NoError(tlsConn.HandshakeContext(serverCtx))
		go func(c net.Conn) { _ = c.Close() }(conn) // Close might block due to a grace period
	}

	for _, serverName := range rejectedServerNames {
		clientTLSConf, err := clientconn.TLSConfig(mtls.CentralSubject, clientconn.TLSConfigOptions{
			ServerName: serverName,
			RootCAs:    getCertPool(mgr.internalTrustRoots),
		})
		if !s.NoError(err) {
			continue
		}
		conn, err := (&net.Dialer{}).DialContext(serverCtx, lis.Addr().Network(), lis.Addr().String())
		if !s.NoError(err) {
			continue
		}
		tlsConn := tls.Client(conn, clientTLSConf)
		s.ErrorAs(tlsConn.HandshakeContext(serverCtx), &x509.HostnameError{})
		go func(c net.Conn) { _ = c.Close() }(conn) // Close might block due to a grace period
	}

	s.Require().NoError(server.Close())
	err = <-serverErrC
	s.ErrorIs(err, net.ErrClosed)
}

func getCertPool(certs []*x509.Certificate) *x509.CertPool {
	pool := x509.NewCertPool()
	for _, cert := range certs {
		pool.AddCert(cert)
	}
	return pool
}
