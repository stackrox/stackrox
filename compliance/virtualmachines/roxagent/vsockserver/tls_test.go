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
	"testing/synctest"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
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

// TestCARefresher_InvalidPEM documents that ensureFreshPool - and therefore
// any handshake relying on it - fails when the fetched bundle contains no
// valid certificates, rather than caching an unusable empty pool.
func TestCARefresher_InvalidPEM(t *testing.T) {
	r := NewCARefresher(
		WithFetchFunc(func(context.Context) ([]byte, error) {
			return []byte("not-a-valid-pem"), nil
		}),
		WithFetchTimeout(time.Second),
	)

	_, err := r.ensureFreshPool(context.Background())
	require.Error(t, err)
	assert.ErrorContains(t, err, "no CA pool cached and fetch failed")
}

func TestCARefresher_HandshakeFailsWhenCAServiceUnavailable(t *testing.T) {
	caPEM, caCert, caKey := testCA(t)

	var available atomic.Bool
	r := NewCARefresher(WithFetchFunc(func(context.Context) ([]byte, error) {
		if !available.Load() {
			return nil, assert.AnError
		}
		return caPEM, nil
	}))

	serverCert := testServerCert(t)
	clientCert := testLeafCert(t, caCert, caKey)
	clientTLS := &tls.Config{Certificates: []tls.Certificate{clientCert}, InsecureSkipVerify: true}

	available.Store(false)
	assert.Error(t, dialTLS(t, r, serverCert, clientTLS), "handshake should fail while the CA service is unavailable")

	available.Store(true)
	assert.NoError(t, dialTLS(t, r, serverCert, clientTLS), "handshake should succeed once the CA service becomes available")
}

// TestCARefresher_ConcurrentHandshakesEachGetValidPool calls
// ensureFreshPool directly from several goroutines at once, bypassing
// server.go's semaphore (which limits the agent to one in-flight
// connection) to exercise the defensive path when multiple connections are
// established in parallel.
func TestCARefresher_ConcurrentHandshakesEachGetValidPool(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		caPEM, caCert, caKey := testCA(t)

		var fetchCount atomic.Int32
		fetchesMayProceed := concurrency.NewSignal()
		r := NewCARefresher(WithFetchFunc(func(context.Context) ([]byte, error) {
			fetchCount.Add(1)
			fetchesMayProceed.Wait()
			return caPEM, nil
		}))

		const callers = 5
		type result struct {
			pool *x509.CertPool
			err  error
		}
		results := make(chan result, callers)
		for range callers {
			go func() {
				pool, err := r.ensureFreshPool(context.Background())
				results <- result{pool: pool, err: err}
			}()
		}

		synctest.Wait()
		fetchesMayProceed.Signal()

		leaf := testLeafCert(t, caCert, caKey)
		leafCert, err := x509.ParseCertificate(leaf.Certificate[0])
		require.NoError(t, err)

		for range callers {
			res := <-results
			require.NoError(t, res.err)
			require.NotNil(t, res.pool, "every caller must get back a valid pool")
			_, verifyErr := leafCert.Verify(x509.VerifyOptions{
				Roots:     res.pool,
				KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			})
			require.NoError(t, verifyErr, "pool returned to caller must be able to verify a cert signed by the fetched CA")
		}
		assert.GreaterOrEqual(t, fetchCount.Load(), int32(1), "at least one caller must have triggered a fetch")

		finalPool, ok := r.cachedPool()
		require.True(t, ok, "cache must be populated after concurrent fetches complete")
		require.NotNil(t, finalPool)
	})
}

// TestCARefresher_Refresh proves that every handshake fetches the latest
// CA. When the CA service starts returning a new bundle, the very next
// handshake picks it up and the old CA is fully rotated out.
func TestCARefresher_Refresh(t *testing.T) {
	ca1PEM, ca1Cert, ca1Key := testCA(t)
	ca2PEM, ca2Cert, ca2Key := testCA(t)

	var callCount atomic.Int32
	r := NewCARefresher(
		WithFetchFunc(func(context.Context) ([]byte, error) {
			n := callCount.Add(1)
			if n == 1 {
				return ca1PEM, nil
			}
			return ca2PEM, nil
		}),
	)

	_, err := r.ensureFreshPool(context.Background())
	require.NoError(t, err)
	require.Equal(t, int32(1), callCount.Load())

	serverCert := testServerCert(t)
	clientCert := testLeafCert(t, ca2Cert, ca2Key)
	doTLSHandshake(t, r, serverCert, clientCert)
	assert.Equal(t, int32(2), callCount.Load(), "handshake must trigger a fresh fetch")

	oldClientCert := testLeafCert(t, ca1Cert, ca1Key)
	err = doTLSHandshakeFails(t, r, serverCert, oldClientCert)
	require.Error(t, err)
}

// TestCARefresher_RefreshFailure_KeepsOldCA proves that a failed fetch
// falls back to the cached pool rather than failing the handshake.
func TestCARefresher_RefreshFailure_KeepsOldCA(t *testing.T) {
	caPEM, caCert, caKey := testCA(t)

	var callCount atomic.Int32
	r := NewCARefresher(
		WithFetchFunc(func(context.Context) ([]byte, error) {
			n := callCount.Add(1)
			if n == 1 {
				return caPEM, nil
			}
			return nil, assert.AnError
		}),
	)

	_, err := r.ensureFreshPool(context.Background())
	require.NoError(t, err)
	require.Equal(t, int32(1), callCount.Load())

	serverCert := testServerCert(t)
	clientCert := testLeafCert(t, caCert, caKey)
	doTLSHandshake(t, r, serverCert, clientCert)
	assert.Equal(t, int32(2), callCount.Load(), "fetch was attempted despite failure")
}

// TestCARefresher_OverlapBundleTrustsBothOldAndNewCA models KubeVirt's real
// rotation behavior: during the overlap window, the CA service returns a
// bundle containing *both* the old and new CA concatenated.
func TestCARefresher_OverlapBundleTrustsBothOldAndNewCA(t *testing.T) {
	oldCAPEM, oldCACert, oldCAKey := testCA(t)
	newCAPEM, newCACert, newCAKey := testCA(t)
	overlapBundle := append(append([]byte{}, oldCAPEM...), newCAPEM...)

	r := NewCARefresher(WithFetchFunc(func(context.Context) ([]byte, error) {
		return overlapBundle, nil
	}))

	serverCert := testServerCert(t)

	oldClientCert := testLeafCert(t, oldCACert, oldCAKey)
	doTLSHandshake(t, r, serverCert, oldClientCert)

	newClientCert := testLeafCert(t, newCACert, newCAKey)
	doTLSHandshake(t, r, serverCert, newClientCert)
}

// TestCARefresher_HardCutover covers KubeVirt hard CA cutover: the CA
// service starts serving a new CA bundle while the cache still holds the
// old one. The always-fetch approach means the very next handshake picks
// up the new CA without any retry logic.
//
// Acceptance criterion (ROX-35848): simulated rotation — the first
// handshake after rotation succeeds immediately.
func TestCARefresher_HardCutover(t *testing.T) {
	ca1PEM, ca1Cert, ca1Key := testCA(t)
	ca2PEM, ca2Cert, ca2Key := testCA(t)

	var fetchCount atomic.Int32
	r := NewCARefresher(WithFetchFunc(func(context.Context) ([]byte, error) {
		n := fetchCount.Add(1)
		if n == 1 {
			return ca1PEM, nil
		}
		return ca2PEM, nil
	}))

	serverCert := testServerCert(t)

	client1 := testLeafCert(t, ca1Cert, ca1Key)
	doTLSHandshake(t, r, serverCert, client1)
	require.Equal(t, int32(1), fetchCount.Load())

	client2 := testLeafCert(t, ca2Cert, ca2Key)
	doTLSHandshake(t, r, serverCert, client2)
	assert.Equal(t, int32(2), fetchCount.Load(),
		"handshake after CA rotation fetches the new CA and succeeds immediately")

	// Old CA must no longer verify — proves replacement, not union.
	err := doTLSHandshakeFails(t, r, serverCert, client1)
	require.Error(t, err)
}

// TestCARefresher_TLSHandshake_ClientCertScenarios covers how the first
// handshake against a cold cache resolves for different client certificate
// presentations. Every case fetches the CA exactly once via
// GetConfigForClient; Go's standard RequireAndVerifyClientCert handles
// verification — there is no custom retry.
func TestCARefresher_TLSHandshake_ClientCertScenarios(t *testing.T) {
	caPEM, caCert, caKey := testCA(t)
	_, wrongCACert, wrongCAKey := testCA(t)
	correctLeaf := testLeafCert(t, caCert, caKey)
	wrongLeaf := testLeafCert(t, wrongCACert, wrongCAKey)

	tests := map[string]struct {
		clientTLS   *tls.Config
		wantErr     bool
		wantFetches int32
	}{
		"cert signed by the current CA succeeds": {
			clientTLS:   &tls.Config{Certificates: []tls.Certificate{correctLeaf}, InsecureSkipVerify: true},
			wantFetches: 1,
		},
		"cert signed by an unrelated CA fails": {
			clientTLS:   &tls.Config{Certificates: []tls.Certificate{wrongLeaf}, InsecureSkipVerify: true},
			wantErr:     true,
			wantFetches: 1,
		},
		"no certificate is rejected": {
			clientTLS:   &tls.Config{InsecureSkipVerify: true},
			wantErr:     true,
			wantFetches: 1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var fetchCount atomic.Int32
			r := NewCARefresher(WithFetchFunc(func(context.Context) ([]byte, error) {
				fetchCount.Add(1)
				return caPEM, nil
			}))

			err := dialTLS(t, r, testServerCert(t), tc.clientTLS)
			if !tc.wantErr {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			assert.Equal(t, tc.wantFetches, fetchCount.Load())
		})
	}
}

// dialTLS runs a server using r's TLS config against a client dialing with
// clientTLS, and returns the server side's handshake result.
func dialTLS(t *testing.T, r *CARefresher, serverCert tls.Certificate, clientTLS *tls.Config) error {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = ln.Close() }()

	tlsCfg := r.TLSConfig(serverCert)

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

	clientConn, dialErr := tls.Dial("tcp", ln.Addr().String(), clientTLS)
	if dialErr == nil {
		_ = clientConn.Close()
	}
	return <-serverErr
}

// doTLSHandshake performs a full TLS handshake between a server using the
// refresher's TLS config and a client presenting the given cert.
func doTLSHandshake(t *testing.T, r *CARefresher, serverCert, clientCert tls.Certificate) {
	t.Helper()
	clientTLS := &tls.Config{Certificates: []tls.Certificate{clientCert}, InsecureSkipVerify: true}
	require.NoError(t, dialTLS(t, r, serverCert, clientTLS), "server handshake should succeed")
}

// doTLSHandshakeFails is doTLSHandshake for the rejection path.
func doTLSHandshakeFails(t *testing.T, r *CARefresher, serverCert, clientCert tls.Certificate) error {
	t.Helper()
	clientTLS := &tls.Config{Certificates: []tls.Certificate{clientCert}, InsecureSkipVerify: true}
	err := dialTLS(t, r, serverCert, clientTLS)
	assert.Error(t, err, "server handshake should fail")
	return err
}

func TestExtractBundleRaw(t *testing.T) {
	payload := []byte("test-ca-pem-bytes")

	tests := map[string]struct {
		input      []byte
		wantRaw    []byte
		wantErrMsg string
	}{
		"well-formed Raw field": {
			input:   protowire.AppendBytes(protowire.AppendTag(nil, 1, protowire.BytesType), payload),
			wantRaw: payload,
		},
		"empty input": {
			input:      nil,
			wantErrMsg: "Bundle.Raw field not found in KubeVirt CA response",
		},
		"unknown varint field skipped before Raw field appears": {
			input: protowire.AppendBytes(
				protowire.AppendTag(
					protowire.AppendVarint(protowire.AppendTag(nil, 2, protowire.VarintType), 5),
					1, protowire.BytesType),
				payload),
			wantRaw: payload,
		},
		"unknown fixed32 field skipped before Raw field appears": {
			input: protowire.AppendBytes(
				protowire.AppendTag(
					protowire.AppendFixed32(protowire.AppendTag(nil, 3, protowire.Fixed32Type), 0xdeadbeef),
					1, protowire.BytesType),
				payload),
			wantRaw: payload,
		},
		"unknown fixed64 field skipped before Raw field appears": {
			input: protowire.AppendBytes(
				protowire.AppendTag(
					protowire.AppendFixed64(protowire.AppendTag(nil, 4, protowire.Fixed64Type), 0xdeadbeefdeadbeef),
					1, protowire.BytesType),
				payload),
			wantRaw: payload,
		},
		"truncated tag (mid-varint cutoff)": {
			input:      []byte{0x80},
			wantErrMsg: "malformed KubeVirt Bundle response",
		},
		"malformed bytes field: length prefix exceeds remaining data": {
			input: append(
				protowire.AppendVarint(protowire.AppendTag(nil, 1, protowire.BytesType), 100),
				[]byte{0x01, 0x02}...,
			),
			wantErrMsg: "malformed protobuf bytes field",
		},
		"malformed varint field": {
			input:      append(protowire.AppendTag(nil, 2, protowire.VarintType), 0x80),
			wantErrMsg: "malformed protobuf varint field",
		},
		"malformed fixed32 field: too few bytes": {
			input:      append(protowire.AppendTag(nil, 3, protowire.Fixed32Type), 0x01, 0x02),
			wantErrMsg: "malformed protobuf fixed32 field",
		},
		"malformed fixed64 field: too few bytes": {
			input:      append(protowire.AppendTag(nil, 4, protowire.Fixed64Type), 0x01, 0x02, 0x03),
			wantErrMsg: "malformed protobuf fixed64 field",
		},
		"unsupported wire type (group)": {
			input:      protowire.AppendTag(nil, 1, protowire.StartGroupType),
			wantErrMsg: "unsupported protobuf wire type",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			raw, err := extractBundleRaw(tc.input)
			if tc.wantErrMsg != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantErrMsg)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantRaw, raw)
		})
	}
}
