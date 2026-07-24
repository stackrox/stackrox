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
	"golang.org/x/time/rate"
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

	serverCert := testServerCert(t)
	clientCert := testLeafCert(t, caCert, caKey)
	clientTLS := &tls.Config{Certificates: []tls.Certificate{clientCert}, InsecureSkipVerify: true}

	// CA service "closed": the handshake must fail, not fall back to some
	// stale/empty pool.
	available.Store(false)
	assert.Error(t, dialTLS(t, r, serverCert, clientTLS), "handshake should fail while the CA service is unavailable")

	// CA service "open": the very next handshake must succeed, with no
	// independent warm-up ever having occurred.
	available.Store(true)
	assert.NoError(t, dialTLS(t, r, serverCert, clientTLS), "handshake should succeed once the CA service becomes available")
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
	err = doTLSHandshakeFails(t, r, serverCert, oldClientCert)
	require.Error(t, err)
	assert.ErrorContains(t, err, "verifying client certificate after KubeVirt CA refresh")
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

// TestCARefresher_HardCutover_ForcesRefetchWhileCacheFresh covers KubeVirt
// hard CA cutover: the cache is still within the age gate (so ensureFreshPool
// would not refetch), but the peer presents a leaf signed by the new CA.
// The handshake must force a CABundle fetch and succeed on the first attempt.
//
// Acceptance criterion (ROX-35848): simulated rotation with a fresh cache —
// the first handshake after rotation force-fetches and succeeds.
func TestCARefresher_HardCutover_ForcesRefetchWhileCacheFresh(t *testing.T) {
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
	// Long cooldown so the second handshake stays rate-limited without wall-clock races.
	r.forceFetchLimiter = rate.NewLimiter(rate.Every(time.Hour), 1)

	_, err := r.ensureFreshPool(context.Background())
	require.NoError(t, err)
	require.Equal(t, int32(1), fetchCount.Load())
	// Deliberately do NOT zero r.fetchedAt — cache must still be "fresh".

	serverCert := testServerCert(t)
	newClientCert := testLeafCert(t, ca2Cert, ca2Key)
	doTLSHandshake(t, r, serverCert, newClientCert)

	assert.Equal(t, int32(2), fetchCount.Load(),
		"verify failure against the cached CA must force exactly one refetch")

	// Forced fetch replaces the pool; old leaves must no longer verify.
	oldClientCert := testLeafCert(t, ca1Cert, ca1Key)
	err = doTLSHandshakeFails(t, r, serverCert, oldClientCert)
	require.Error(t, err)
	assert.ErrorContains(t, err, "must re-fetch KubeVirt CA, but was rate-limited")
}

// TestCARefresher_HardCutover_ForceRefetchFailureKeepsRejecting proves the
// force path does not fall back to the stale pool: that pool already
// failed to verify the peer once, so reusing it would just accept the same
// unverifiable peer.
//
// Acceptance criterion (ROX-35848): a forced fetch failure does not accept
// the peer.
func TestCARefresher_HardCutover_ForceRefetchFailureKeepsRejecting(t *testing.T) {
	ca1PEM, _, _ := testCA(t)
	_, ca2Cert, ca2Key := testCA(t)

	var fetchCount atomic.Int32
	r := NewCARefresher(WithFetchFunc(func(context.Context) ([]byte, error) {
		n := fetchCount.Add(1)
		if n == 1 {
			return ca1PEM, nil
		}
		return nil, assert.AnError
	}))

	_, err := r.ensureFreshPool(context.Background())
	require.NoError(t, err)

	serverCert := testServerCert(t)
	newClientCert := testLeafCert(t, ca2Cert, ca2Key)
	err = doTLSHandshakeFails(t, r, serverCert, newClientCert)
	require.Error(t, err)
	assert.ErrorContains(t, err, "must re-fetch KubeVirt CA, but re-fetch also failed")
	assert.Equal(t, int32(2), fetchCount.Load(), "verify failure must still attempt a forced refetch")
}

// TestCARefresher_FreshCache_SecondHandshakeDoesNotRefetch locks case 1:
// after a successful warm, a still-fresh cache must not hit CABundle again
// on the next successful handshake.
//
// Acceptance criterion (ROX-35848): the happy path still uses the cache —
// no fetch on every handshake.
func TestCARefresher_FreshCache_SecondHandshakeDoesNotRefetch(t *testing.T) {
	caPEM, caCert, caKey := testCA(t)

	var fetchCount atomic.Int32
	r := NewCARefresher(WithFetchFunc(func(context.Context) ([]byte, error) {
		fetchCount.Add(1)
		return caPEM, nil
	}))

	serverCert := testServerCert(t)
	clientCert := testLeafCert(t, caCert, caKey)

	doTLSHandshake(t, r, serverCert, clientCert)
	require.Equal(t, int32(1), fetchCount.Load())

	doTLSHandshake(t, r, serverCert, clientCert)
	assert.Equal(t, int32(1), fetchCount.Load(),
		"successful verify against a fresh cache must not refetch")
}

// TestCARefresher_TLSHandshake_ClientCertScenarios covers how the very
// first handshake against a cold cache resolves for different client
// certificate presentations. Every case fetches the CA exactly once via
// GetConfigForClient before the client's certificate is ever inspected;
// the "wrong CA" case fetches a second time via verifyPeerCertificate's
// forced retry.
//
// Acceptance criterion (ROX-35848): a peer that verifies against neither the
// old nor the new CA still fails, with only a single forced retry.
func TestCARefresher_TLSHandshake_ClientCertScenarios(t *testing.T) {
	caPEM, caCert, caKey := testCA(t)
	_, wrongCACert, wrongCAKey := testCA(t)
	correctLeaf := testLeafCert(t, caCert, caKey)
	wrongLeaf := testLeafCert(t, wrongCACert, wrongCAKey)

	tests := map[string]struct {
		clientTLS *tls.Config
		wantErr   bool
		// wantErrMsg, if set, is asserted with ErrorContains; leave empty to
		// only require that some error occurred (e.g. wording owned by
		// crypto/tls itself, not this package).
		wantErrMsg  string
		wantFetches int32
	}{
		"cert signed by the current CA succeeds": {
			clientTLS:   &tls.Config{Certificates: []tls.Certificate{correctLeaf}, InsecureSkipVerify: true},
			wantFetches: 1,
		},
		"cert signed by an unrelated CA fails after one forced retry": {
			clientTLS:   &tls.Config{Certificates: []tls.Certificate{wrongLeaf}, InsecureSkipVerify: true},
			wantErr:     true,
			wantErrMsg:  "verifying client certificate after KubeVirt CA refresh",
			wantFetches: 2,
		},
		"no certificate is rejected by crypto/tls before verifyPeerCertificate runs": {
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
				if tc.wantErrMsg != "" {
					assert.ErrorContains(t, err, tc.wantErrMsg)
				}
			}
			assert.Equal(t, tc.wantFetches, fetchCount.Load())
		})
	}
}

// TestCARefresher_ForceFetchCooldown proves the forced refetch in
// verifyPeerCertificate is rate-limited, and that the limit is a cooldown
// rather than a permanent lockout: a second verify failure inside the
// cooldown window is rejected without a new fetch, while one arriving after
// the cooldown elapses triggers another forced fetch.
func TestCARefresher_ForceFetchCooldown(t *testing.T) {
	tests := map[string]struct {
		// limiter overrides forceFetchLimiter; nil keeps defaultForceFetchCooldown,
		// which this test then has to really wait out.
		limiter        *rate.Limiter
		wait           time.Duration
		wantSecondMsg  string
		wantFetchCount int32
	}{
		"repeated failure inside the cooldown is rate-limited, not re-fetched": {
			limiter:        rate.NewLimiter(rate.Every(time.Hour), 1),
			wantSecondMsg:  "must re-fetch KubeVirt CA, but was rate-limited",
			wantFetchCount: 2,
		},
		"failure after the cooldown elapses triggers another forced fetch": {
			wait:           defaultForceFetchCooldown + 500*time.Millisecond,
			wantSecondMsg:  "verifying client certificate after KubeVirt CA refresh",
			wantFetchCount: 3,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			caPEM, _, _ := testCA(t)
			_, wrongCACert, wrongCAKey := testCA(t)

			var fetchCount atomic.Int32
			r := NewCARefresher(WithFetchFunc(func(context.Context) ([]byte, error) {
				fetchCount.Add(1)
				return caPEM, nil
			}))
			if tc.limiter != nil {
				r.forceFetchLimiter = tc.limiter
			}

			serverCert := testServerCert(t)
			wrongClientCert := testLeafCert(t, wrongCACert, wrongCAKey)

			err := doTLSHandshakeFails(t, r, serverCert, wrongClientCert)
			require.Error(t, err)
			require.Equal(t, int32(2), fetchCount.Load(), "cold ensureFreshPool + one forced refetch")

			if tc.wait > 0 {
				// Can't use synctest: dialTLS uses a real TCP loopback
				// socket, and synctest never treats network I/O as durably
				// blocked.
				time.Sleep(tc.wait)
			}

			err = doTLSHandshakeFails(t, r, serverCert, wrongClientCert)
			require.Error(t, err)
			assert.ErrorContains(t, err, tc.wantSecondMsg)
			assert.Equal(t, tc.wantFetchCount, fetchCount.Load())
		})
	}
}

// dialTLS runs a server using r's TLS config against a client dialing with
// clientTLS, and returns the server side's handshake result. It is the
// shared plumbing under doTLSHandshake, doTLSHandshakeFails, and any test
// that needs a client TLS config dialTLS itself can't assume (e.g. one
// presenting no certificate at all).
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
