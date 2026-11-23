package tlsprofile

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/fips140"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"math/big"
	"net"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type profileSuite struct {
	suite.Suite
}

func TestProfileSuite(t *testing.T) {
	suite.Run(t, new(profileSuite))
}

func (s *profileSuite) SetupSubTest() {
	minVersionOnce = sync.Once{}
	cipherSuitesOnce = sync.Once{}
	curvePreferencesOnce = sync.Once{}
	cachedCurvePreferences = nil
}

func (s *profileSuite) TestParseMinVersion() {
	tests := []struct {
		input   string
		want    uint16
		wantErr bool
	}{
		{"TLSv1", tls.VersionTLS10, false},
		{"TLSv1.1", tls.VersionTLS11, false},
		{"TLSv1.2", tls.VersionTLS12, false},
		{"TLSv1.3", tls.VersionTLS13, false},
		{" TLSv1.3 ", tls.VersionTLS13, false},
		{"", 0, true},
		{"1.2", 0, true},
		{"VersionTLS12", 0, true},
		{"TLS1.2", 0, true},
		{"ssl3", 0, true},
	}
	for _, tt := range tests {
		s.Run(tt.input, func() {
			got, err := parseMinVersion(tt.input)
			if tt.wantErr {
				assert.Error(s.T(), err)
			} else {
				require.NoError(s.T(), err)
				assert.Equal(s.T(), tt.want, got)
			}
		})
	}
}

func (s *profileSuite) TestParseCipherSuites() {
	tests := []struct {
		name    string
		input   string
		want    []uint16
		wantErr bool
	}{
		{
			name:  "single suite",
			input: "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
			want:  []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384},
		},
		{
			name:  "multiple suites",
			input: "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
			want: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			},
		},
		{
			name:  "whitespace around names",
			input: " TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384 , TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256 ",
			want: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			},
		},
		{
			name:  "trailing comma ignored",
			input: "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,",
			want:  []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384},
		},
		{
			name:    "all unknown suites",
			input:   "TLS_DOES_NOT_EXIST",
			wantErr: true,
		},
		{
			name:  "unknown suites are skipped",
			input: "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,TLS_DOES_NOT_EXIST,TLS_RSA_WITH_3DES_EDE_CBC_SHA",
			want:  []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384},
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "only commas",
			input:   ",,,",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := parseCipherSuites(tt.input)
			if tt.wantErr {
				assert.Error(s.T(), err)
			} else {
				require.NoError(s.T(), err)
				assert.Equal(s.T(), tt.want, got)
			}
		})
	}
}

func (s *profileSuite) TestParseGroups() {
	tests := []struct {
		name    string
		input   string
		want    []tls.CurveID
		wantErr bool
	}{
		{
			name:  "single group",
			input: "X25519",
			want:  []tls.CurveID{tls.X25519},
		},
		{
			name:  "multiple groups preserve order",
			input: "X25519MLKEM768,secp256r1",
			want:  []tls.CurveID{tls.X25519MLKEM768, tls.CurveP256},
		},
		{
			name:  "whitespace and trailing comma",
			input: " secp384r1 , secp521r1 ,",
			want:  []tls.CurveID{tls.CurveP384, tls.CurveP521},
		},
		{
			name:  "unknown groups are skipped",
			input: "secp256r1,bogus",
			want:  []tls.CurveID{tls.CurveP256},
		},
		{
			name:    "all unknown",
			input:   "bogus",
			wantErr: true,
		},
		{
			name:    "empty",
			input:   "",
			wantErr: true,
		},
		{
			name:    "only commas",
			input:   ",,,",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := parseGroups(tt.input)
			if tt.wantErr {
				assert.Error(s.T(), err)
			} else {
				require.NoError(s.T(), err)
				assert.Equal(s.T(), tt.want, got)
			}
		})
	}
}

func (s *profileSuite) TestCurvePreferences() {
	for _, tt := range []struct {
		name string
		env  string
		want []tls.CurveID
	}{
		{"nil when unset", "", nil},
		{"explicit groups", "X25519,secp256r1", []tls.CurveID{tls.X25519, tls.CurveP256}},
		{"invalid falls back to nil", "bogus", nil},
	} {
		s.Run(tt.name, func() {
			s.T().Setenv("ROX_TLS_GROUPS", tt.env)
			s.Equal(tt.want, CurvePreferences())
		})
	}
}

func (s *profileSuite) TestMinVersion() {
	for _, tt := range []struct {
		name string
		env  string
		want uint16
	}{
		{"default when unset", "", defaultMinVersion},
		{"explicit TLSv1.3", "TLSv1.3", tls.VersionTLS13},
		{"invalid falls back to default", "bogus", defaultMinVersion},
	} {
		s.Run(tt.name, func() {
			s.T().Setenv("ROX_TLS_MIN_VERSION", tt.env)
			s.Equal(tt.want, MinVersion())
		})
	}
}

func (s *profileSuite) TestCipherSuites() {
	for _, tt := range []struct {
		name string
		env  string
		want []uint16
	}{
		{"default when unset", "", defaultCipherSuites},
		{"explicit single suite", "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384", []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384}},
		{"invalid falls back to default", "NOT_A_CIPHER", defaultCipherSuites},
	} {
		s.Run(tt.name, func() {
			s.T().Setenv("ROX_TLS_CIPHER_SUITES", tt.env)
			s.Equal(tt.want, CipherSuites())
		})
	}
}

func TestCurvePreferencesFIPSHandshake(t *testing.T) {
	for name, tc := range map[string]struct {
		mode        string
		groups      string
		wantFailure bool
	}{
		"x25519 without FIPS": {"off", "X25519", false},
		"x25519 with FIPS":    {"on", "X25519", true},
		"p256 with FIPS":      {"on", "secp256r1", false},
	} {
		if child := os.Getenv("ROX_TEST_TLS_HANDSHAKE_CASE"); child != "" {
			if child == name {
				require.Equal(t, tc.mode == "on", fips140.Enabled())
				checkProfileHandshake(t, tc.wantFailure)
				return
			}
			continue
		}
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCurvePreferencesFIPSHandshake$")
			cmd.Env = append(os.Environ(), "ROX_TEST_TLS_HANDSHAKE_CASE="+name,
				"GODEBUG=fips140="+tc.mode, "ROX_TLS_GROUPS="+tc.groups)
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "%s", output)
		})
	}
	require.Empty(t, os.Getenv("ROX_TEST_TLS_HANDSHAKE_CASE"), "unknown handshake case")
}

func checkProfileHandshake(t *testing.T, wantFailure bool) {
	t.Helper()
	const hostname = "tlsprofile.test"
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		DNSNames:     []string{hostname},
		NotBefore:    now.Add(-time.Minute),
		NotAfter:     now.Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	cert := tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
	leaf, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	roots := x509.NewCertPool()
	roots.AddCert(leaf)

	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() { assert.NoError(t, serverConn.Close()) })
	t.Cleanup(func() { assert.NoError(t, clientConn.Close()) })
	deadline := time.Now().Add(5 * time.Second)
	require.NoError(t, serverConn.SetDeadline(deadline))
	require.NoError(t, clientConn.SetDeadline(deadline))

	server := tls.Server(serverConn, &tls.Config{
		MinVersion:       tls.VersionTLS13,
		Certificates:     []tls.Certificate{cert},
		CurvePreferences: CurvePreferences(),
	})
	client := tls.Client(clientConn, &tls.Config{
		MinVersion: tls.VersionTLS13,
		ServerName: hostname,
		RootCAs:    roots,
		// Offer a key share usable by each successful case without a retry.
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
	})
	serverResult := make(chan error, 1)
	go func() { serverResult <- server.Handshake() }()
	clientErr := client.Handshake()
	serverErr := <-serverResult
	if wantFailure {
		require.ErrorContains(t, serverErr, "no key exchanges supported by both client and server")
		require.Error(t, clientErr)
		return
	}
	require.NoError(t, clientErr, "server handshake error: %v", serverErr)
	require.NoError(t, serverErr)
}
