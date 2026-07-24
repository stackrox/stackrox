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
	assert.ErrorContains(t, err, "no valid certificates")
}

func TestCARefresher_HandshakeFetchesOnColdCache(t *testing.T) {
	caPEM, caCert, caKey := testCA(t)

	var fetchCount atomic.Int32
	r := NewCARefresher(WithFetchFunc(func(context.Context) ([]byte, error) {
		fetchCount.Add(1)
		return caPEM, nil
	}))

	// The cache starts out completely cold, mirroring roxagent immediately
	// after boot in KubeVirt's namespace-isolated VSOCK mode, where the CA
	// service does not exist until a real connection arrives. What's under
	// test is that the handshake itself - not any independent warm-up -
	// is what populates the cache.
	serverCert := testServerCert(t)
	clientCert := testLeafCert(t, caCert, caKey)
	doTLSHandshake(t, r, serverCert, clientCert)

	assert.Equal(t, int32(1), fetchCount.Load(), "the handshake should have triggered exactly one fetch")
}

func TestCARefresher_HandshakeFailsWhenCAServiceUnavailable(t *testing.T) {
	caPEM, caCert, caKey := testCA(t)

	// available toggles whether the fake CA service is reachable, modeling
	// KubeVirt's on-demand System.CABundle service existing only for the
	// duration of a virt-handler dial+handshake.
	var available atomic.Bool
	r := NewCARefresher(WithFetchFunc(func(context.Context) ([]byte, error) {
		if !available.Load() {
			return nil, assert.AnError
		}
		return caPEM, nil
	}))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = ln.Close() }()

	tlsCfg := r.TLSConfig(testServerCert(t))
	clientCert := testLeafCert(t, caCert, caKey)
	clientTLS := &tls.Config{Certificates: []tls.Certificate{clientCert}, InsecureSkipVerify: true}

	dial := func() error {
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

	// CA service "closed": the handshake must fail, not fall back to some
	// stale/empty pool.
	available.Store(false)
	assert.Error(t, dial(), "handshake should fail while the CA service is unavailable")

	// CA service "open": the very next handshake must succeed, with no
	// independent warm-up ever having occurred.
	available.Store(true)
	assert.NoError(t, dial(), "handshake should succeed once the CA service becomes available")
}

// TestCARefresher_ConcurrentHandshakesEachGetValidPool calls
// ensureFreshPool directly from several goroutines at once, bypassing
// server.go's semaphore (which limits the agent to one in-flight
// connection) to exercise the defensive path when multiple connections are established in parallel.
func TestCARefresher_ConcurrentHandshakesEachGetValidPool(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		caPEM, caCert, caKey := testCA(t)

		var fetchCount atomic.Int32
		fetchesMayProceed := concurrency.NewSignal()
		r := NewCARefresher(WithFetchFunc(func(context.Context) ([]byte, error) {
			fetchCount.Add(1)
			fetchesMayProceed.Wait() // block until the test says all callers have arrived
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

		synctest.Wait() // Wait until all callers are blocked on fetchesMayProceed.Wait
		fetchesMayProceed.Signal()

		// A leaf cert signed by the fetched CA must verify against every
		// pool handed back. Corrupted pool would fail this assertion.
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

		finalPool, _, ok := r.cachedPool()
		require.True(t, ok, "cache must be populated after concurrent fetches complete")
		require.NotNil(t, finalPool)
	})
}

// TestCARefresher_Refresh proves that a stale cache is picked up and fully
// rotated by the handshake path itself - there is no background loop left
// to drive a second fetch, so ensureFreshPool being called from
// GetConfigForClient during a real handshake is the only thing that can.
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
	// Warm the cache with CA1 via a direct call.
	_, err := r.ensureFreshPool(context.Background())
	require.NoError(t, err)
	require.Equal(t, int32(1), callCount.Load())

	// Force cache stale.
	r.fetchedAt = time.Time{}

	// The second CA is now active, fetched by the handshake below.
	serverCert := testServerCert(t)
	clientCert := testLeafCert(t, ca2Cert, ca2Key)
	doTLSHandshake(t, r, serverCert, clientCert)
	assert.Equal(t, int32(2), callCount.Load(), "the stale handshake should have triggered exactly one refetch")

	// The first CA was fully rotated out (not merely appended to): a
	// refresher bug that unioned pools instead of replacing them would pass
	// unnoticed without this assertion.
	oldClientCert := testLeafCert(t, ca1Cert, ca1Key)
	doTLSHandshakeFails(t, r, serverCert, oldClientCert)
}

// TestCARefresher_RefreshFailure_KeepsOldCA proves that a failed refetch,
// triggered by a stale-cache handshake, falls back to the last known-good
// CA rather than failing the handshake outright.
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
	// Warm the cache with the original CA via a direct call.
	_, err := r.ensureFreshPool(context.Background())
	require.NoError(t, err)
	require.Equal(t, int32(1), callCount.Load())

	// Force cache stale.
	r.fetchedAt = time.Time{}

	// Original CA should still work.
	serverCert := testServerCert(t)
	clientCert := testLeafCert(t, caCert, caKey)
	doTLSHandshake(t, r, serverCert, clientCert)
	assert.Equal(t, int32(2), callCount.Load(), "the stale handshake should have attempted a refetch")
}

// TestCARefresher_StaleCacheTriggersRefetchDuringHandshake proves the
// minimal case: a stale cache is refetched by ensureFreshPool being called
// from GetConfigForClient during a real handshake. TestCARefresher_Refresh
// covers the same mechanism plus full CA rotation (old CA fully replaced,
// not unioned with the new one).
func TestCARefresher_StaleCacheTriggersRefetchDuringHandshake(t *testing.T) {
	ca1PEM, _, _ := testCA(t)
	ca2PEM, ca2Cert, ca2Key := testCA(t)

	var fetchCount atomic.Int32
	r := NewCARefresher(
		WithFetchFunc(func(context.Context) ([]byte, error) {
			if fetchCount.Add(1) == 1 {
				return ca1PEM, nil
			}
			return ca2PEM, nil
		}),
	)
	// Warm the cache with CA1 via a direct call.
	_, err := r.ensureFreshPool(context.Background())
	require.NoError(t, err)
	require.Equal(t, int32(1), fetchCount.Load())

	// Force cache stale.
	r.fetchedAt = time.Time{}

	// The next handshake presents a CA2-signed client cert. It can only
	// succeed if the handshake path itself noticed the staleness and
	// fetched a replacement pool inline.
	serverCert := testServerCert(t)
	clientCert := testLeafCert(t, ca2Cert, ca2Key)
	doTLSHandshake(t, r, serverCert, clientCert)

	assert.Equal(t, int32(2), fetchCount.Load(), "the stale handshake should have triggered exactly one refetch")
}

// TestCARefresher_OverlapBundleTrustsBothOldAndNewCA models KubeVirt's real
// rotation behavior: during the overlap window, the CA service returns a
// bundle containing *both* the old and new CA concatenated (not a hard swap
// from one to the other), specifically so nothing already holding an
// old-CA-signed cert breaks mid-rotation. AppendCertsFromPEM and
// x509.CertPool both natively support multiple CAs in one bundle, so this is
// mostly a regression/documentation test rather than new capability.
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

func TestCARefresher_TLSHandshake(t *testing.T) {
	caPEM, caCert, caKey := testCA(t)

	r := NewCARefresher(
		WithFetchFunc(func(context.Context) ([]byte, error) { return caPEM, nil }),
	)

	serverCert := testServerCert(t)
	clientCert := testLeafCert(t, caCert, caKey)
	doTLSHandshake(t, r, serverCert, clientCert)
}

func TestCARefresher_TLSHandshake_WrongCA(t *testing.T) {
	caPEM, _, _ := testCA(t)
	_, wrongCACert, wrongCAKey := testCA(t)

	r := NewCARefresher(
		WithFetchFunc(func(context.Context) ([]byte, error) { return caPEM, nil }),
	)

	serverCert := testServerCert(t)
	wrongClientCert := testLeafCert(t, wrongCACert, wrongCAKey)
	doTLSHandshakeFails(t, r, serverCert, wrongClientCert)
}

// doTLSHandshake performs a full TLS handshake between a server using the
// refresher's TLS config and a client presenting the given cert.
func doTLSHandshake(t *testing.T, r *CARefresher, serverCert, clientCert tls.Certificate) {
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

	clientTLS := &tls.Config{
		Certificates:       []tls.Certificate{clientCert},
		InsecureSkipVerify: true,
	}
	clientConn, err := tls.Dial("tcp", ln.Addr().String(), clientTLS)
	require.NoError(t, err, "client dial should succeed")
	_ = clientConn.Close()

	require.NoError(t, <-serverErr, "server handshake should succeed")
}

// doTLSHandshakeFails performs a TLS handshake between a server using the
// refresher's TLS config and a client presenting the given cert, and asserts
// that the server-side handshake is rejected (e.g. because clientCert isn't
// signed by any CA in the refresher's currently active pool).
func doTLSHandshakeFails(t *testing.T, r *CARefresher, serverCert, clientCert tls.Certificate) {
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
		serverErr <- tlsConn.Handshake()
		_ = tlsConn.Close()
	}()

	clientTLS := &tls.Config{
		Certificates:       []tls.Certificate{clientCert},
		InsecureSkipVerify: true,
	}
	clientConn, err := tls.Dial("tcp", ln.Addr().String(), clientTLS)
	if err == nil {
		_ = clientConn.Close()
	}

	assert.Error(t, <-serverErr, "server handshake should fail")
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
			// 0x80 has its continuation bit set but no follow-up byte, so
			// ConsumeTag's underlying varint read can never terminate.
			input:      []byte{0x80},
			wantErrMsg: "malformed KubeVirt Bundle response",
		},
		"malformed bytes field: length prefix exceeds remaining data": {
			input: append(
				protowire.AppendVarint(protowire.AppendTag(nil, 1, protowire.BytesType), 100),
				[]byte{0x01, 0x02}..., // far fewer than the 100 bytes the length prefix claims
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
