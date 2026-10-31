package tlsprofile

import (
	"crypto/tls"
	"testing"

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
