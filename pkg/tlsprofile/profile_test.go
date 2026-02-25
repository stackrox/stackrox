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
}

func (s *profileSuite) TestParseMinVersion() {
	tests := []struct {
		input   string
		want    uint16
		wantErr bool
	}{
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
			name:    "unknown suite",
			input:   "TLS_DOES_NOT_EXIST",
			wantErr: true,
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
