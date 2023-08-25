package x509utils

import (
	"crypto/x509"
	_ "embed"
	"strings"
	"testing"
	"time"

	"github.com/cloudflare/cfssl/helpers"
	"github.com/stackrox/rox/pkg/timeutil"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	//go:embed testdata/cert-chain.pem
	pemChain string

	//go:embed testdata/verification-time
	verificationTimeStr string

	verificationTime = timeutil.MustParse(time.RFC3339, strings.TrimSpace(verificationTimeStr))

	certChain []*x509.Certificate
	derChain  [][]byte
)

func init() {
	var err error
	certChain, err = helpers.ParseCertificatesPEM([]byte(pemChain))
	utils.CrashOnError(err)

	for _, cert := range certChain {
		derChain = append(derChain, cert.Raw)
	}
}

func TestParseChain(t *testing.T) {
	certChain, err := ParseCertificateChain(derChain)
	require.NoError(t, err)
	require.Len(t, certChain, len(derChain))

	for i, parsedCert := range certChain {
		assert.True(t, parsedCert.Equal(certChain[i]))
	}
}

func TestVerifyChain_VerifyWithRoot(t *testing.T) {
	rootCAs := x509.NewCertPool()
	rootCAs.AddCert(certChain[len(certChain)-1])

	opts := x509.VerifyOptions{
		Roots:       rootCAs,
		CurrentTime: verificationTime,
	}
	assert.NoError(t, VerifyCertificateChain(certChain, opts))
}

func TestVerifyChain_VerifyWithoutRoot(t *testing.T) {
	rootCAs := x509.NewCertPool()
	rootCAs.AddCert(certChain[len(certChain)-1])

	opts := x509.VerifyOptions{
		Roots:       rootCAs,
		CurrentTime: verificationTime,
	}
	assert.NoError(t, VerifyCertificateChain(certChain[:len(certChain)-1], opts))
}

func TestVerifyChain_VerifyWithoutIntermediateFails(t *testing.T) {
	rootCAs := x509.NewCertPool()
	rootCAs.AddCert(certChain[len(certChain)-1])

	opts := x509.VerifyOptions{
		Roots:       rootCAs,
		CurrentTime: verificationTime,
	}
	assert.Error(t, VerifyCertificateChain(certChain[:len(certChain)-2], opts))
}

func TestVerifyChain_VerifyPresetIntermediateIsIgnored(t *testing.T) {
	rootCAs := x509.NewCertPool()
	rootCAs.AddCert(certChain[len(certChain)-1])

	intermediateCAs := x509.NewCertPool()
	intermediateCAs.AddCert(certChain[len(certChain)-2])

	opts := x509.VerifyOptions{
		Roots:       rootCAs,
		CurrentTime: verificationTime,
	}
	assert.Error(t, VerifyCertificateChain(certChain[:len(certChain)-2], opts))
}
