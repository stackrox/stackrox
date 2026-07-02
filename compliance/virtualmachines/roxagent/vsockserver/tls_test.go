package vsockserver

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testCA generates a self-signed CA cert + key and returns the PEM-encoded cert
// along with the parsed structures.
func testCA(t *testing.T) (caPEM []byte, caCert *x509.Certificate, caKey *ecdsa.PrivateKey) {
	t.Helper()
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &caKey.PublicKey, caKey)
	require.NoError(t, err)

	caCert, err = x509.ParseCertificate(certDER)
	require.NoError(t, err)

	caPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	return caPEM, caCert, caKey
}

// testLeafCert creates a leaf certificate signed by the given CA.
func testLeafCert(t *testing.T, caCert *x509.Certificate, caKey *ecdsa.PrivateKey) tls.Certificate {
	t.Helper()
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "test-leaf"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, caCert, &leafKey.PublicKey, caKey)
	require.NoError(t, err)

	return tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  leafKey,
	}
}

// testServerCert creates a self-signed server certificate for TLS listener.
func testServerCert(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(100),
		Subject:      pkix.Name{CommonName: "test-server"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	return tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
	}
}

func TestCARefresher_InitialFetch(t *testing.T) {
	caPEM, _, _ := testCA(t)

	r := NewCARefresher(
		WithFetchFunc(func() ([]byte, error) { return caPEM, nil }),
		WithRefreshInterval(time.Hour),
	)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- r.Start(ctx) }()

	// Give Start time to complete the initial fetch.
	time.Sleep(50 * time.Millisecond)

	cfg := r.TLSConfig()
	require.NotNil(t, cfg)
	assert.Equal(t, tls.RequireAndVerifyClientCert, cfg.ClientAuth)
	assert.Equal(t, uint16(tls.VersionTLS12), cfg.MinVersion)

	// GetConfigForClient should return a config with the CA pool.
	inner, err := cfg.GetConfigForClient(&tls.ClientHelloInfo{})
	require.NoError(t, err)
	require.NotNil(t, inner)
	assert.NotNil(t, inner.ClientCAs)

	cancel()
	require.NoError(t, <-errCh)
}

func TestCARefresher_FetchFailure(t *testing.T) {
	r := NewCARefresher(
		WithFetchFunc(func() ([]byte, error) {
			return nil, assert.AnError
		}),
	)

	err := r.Start(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, assert.AnError)
}

func TestCARefresher_InvalidPEM(t *testing.T) {
	r := NewCARefresher(
		WithFetchFunc(func() ([]byte, error) {
			return []byte("not-a-valid-pem"), nil
		}),
	)

	err := r.Start(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no valid certificates")
}

func TestCARefresher_Refresh(t *testing.T) {
	ca1PEM, _, _ := testCA(t)
	ca2PEM, ca2Cert, ca2Key := testCA(t)

	var callCount atomic.Int32
	r := NewCARefresher(
		WithFetchFunc(func() ([]byte, error) {
			n := callCount.Add(1)
			if n == 1 {
				return ca1PEM, nil
			}
			return ca2PEM, nil
		}),
		WithRefreshInterval(20*time.Millisecond),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- r.Start(ctx) }()

	// Wait for at least one refresh cycle.
	time.Sleep(100 * time.Millisecond)

	assert.GreaterOrEqual(t, callCount.Load(), int32(2), "should have refreshed at least once")

	// Verify the second CA is now active by doing a TLS handshake.
	serverCert := testServerCert(t)
	clientCert := testLeafCert(t, ca2Cert, ca2Key)
	doTLSHandshake(t, r, serverCert, clientCert)

	cancel()
	require.NoError(t, <-errCh)
}

func TestCARefresher_RefreshFailure_KeepsOldCA(t *testing.T) {
	caPEM, caCert, caKey := testCA(t)

	var callCount atomic.Int32
	r := NewCARefresher(
		WithFetchFunc(func() ([]byte, error) {
			n := callCount.Add(1)
			if n == 1 {
				return caPEM, nil
			}
			return nil, assert.AnError
		}),
		WithRefreshInterval(20*time.Millisecond),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- r.Start(ctx) }()

	// Wait for refresh failure.
	time.Sleep(100 * time.Millisecond)

	assert.GreaterOrEqual(t, callCount.Load(), int32(2))

	// Original CA should still work.
	serverCert := testServerCert(t)
	clientCert := testLeafCert(t, caCert, caKey)
	doTLSHandshake(t, r, serverCert, clientCert)

	cancel()
	require.NoError(t, <-errCh)
}

func TestCARefresher_TLSHandshake(t *testing.T) {
	caPEM, caCert, caKey := testCA(t)

	r := NewCARefresher(
		WithFetchFunc(func() ([]byte, error) { return caPEM, nil }),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- r.Start(ctx) }()
	time.Sleep(50 * time.Millisecond)

	serverCert := testServerCert(t)
	clientCert := testLeafCert(t, caCert, caKey)
	doTLSHandshake(t, r, serverCert, clientCert)

	cancel()
	require.NoError(t, <-errCh)
}

func TestCARefresher_TLSHandshake_WrongCA(t *testing.T) {
	caPEM, _, _ := testCA(t)
	_, wrongCACert, wrongCAKey := testCA(t)

	r := NewCARefresher(
		WithFetchFunc(func() ([]byte, error) { return caPEM, nil }),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- r.Start(ctx) }()
	time.Sleep(50 * time.Millisecond)

	serverCert := testServerCert(t)
	wrongClientCert := testLeafCert(t, wrongCACert, wrongCAKey)

	// Handshake should fail: client cert signed by wrong CA.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = ln.Close() }()

	tlsCfg := r.TLSConfig()
	tlsCfg.Certificates = []tls.Certificate{serverCert}

	serverErr := make(chan error, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		tlsConn := tls.Server(conn, tlsCfg)
		serverErr <- tlsConn.Handshake()
		_ = tlsConn.Close()
	}()

	clientTLS := &tls.Config{
		Certificates:       []tls.Certificate{wrongClientCert},
		InsecureSkipVerify: true,
	}
	clientConn, err := tls.Dial("tcp", ln.Addr().String(), clientTLS)
	if err == nil {
		_ = clientConn.Close()
	}

	sErr := <-serverErr
	assert.Error(t, sErr, "handshake should fail with wrong CA")

	cancel()
	require.NoError(t, <-errCh)
}

// doTLSHandshake performs a full TLS handshake between a server using the
// refresher's TLS config and a client presenting the given cert.
func doTLSHandshake(t *testing.T, r *CARefresher, serverCert, clientCert tls.Certificate) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = ln.Close() }()

	tlsCfg := r.TLSConfig()
	tlsCfg.Certificates = []tls.Certificate{serverCert}

	serverErr := make(chan error, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		tlsConn := tls.Server(conn, tlsCfg)
		err = tlsConn.Handshake()
		_ = tlsConn.Close()
		serverErr <- err
	}()

	clientTLS := &tls.Config{
		Certificates:       []tls.Certificate{clientCert},
		InsecureSkipVerify: true,
	}
	clientConn, err := tls.Dial("tcp", ln.Addr().String(), clientTLS)
	require.NoError(t, err, "client dial should succeed")
	_ = clientConn.Close()

	require.NoError(t, <-serverErr, "server handshake should succeed")
}
