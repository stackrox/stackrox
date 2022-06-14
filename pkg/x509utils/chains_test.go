package x509utils

import (
	"crypto/x509"
	"testing"
	"time"

	"github.com/cloudflare/cfssl/helpers"
	"github.com/stackrox/stackrox/pkg/timeutil"
	"github.com/stackrox/stackrox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	pemChain = `
-----BEGIN CERTIFICATE-----
MIICzjCCAbYCCQC9lhw+ZnRUVDANBgkqhkiG9w0BAQUFADAhMR8wHQYDVQQDDBZJ
bnRlcm1lZGlhdGUgU2VydmVyIENBMB4XDTIwMDMwMzE2NTczM1oXDTIwMDQwMjE2
NTczM1owMTEvMC0GA1UEAwwmY3VzdG9tLXRscy1jZXJ0LmNlbnRyYWwuc3RhY2ty
b3gubG9jYWwwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDGQBdYGH63
ZNTS0uyZa0Oha0C3KixjtrF9KR9YZ87JH/1omCgmcJvsB+/C+iuDuPnxSrMD9thR
iCuj15xPccYY1r7LeXk/F092ojRV0zifgHweO3UMOgXdlOxNOpaBefdh5uuVk9V+
1zTrtO/lWUevAmF4/cyRD8QkcFgDi4rJsQm+lWmLZwgiGNvJ7IXPPsuc2anwtifN
rI2hqhm6uZEEgV3CU9VpRUq16QpQSn8J0iyftOeHD11j5RoV5M7+EDyVY14IKlJW
Z8LgnX1H1bIWzjV4yk/ZmSA5gVpslM0uwwZLaHKlihk11IpKSC/CFn6rg33wAOa6
ORS0HEcJgeXrAgMBAAEwDQYJKoZIhvcNAQEFBQADggEBAJfmOwWtf5EA8GMRO21r
WUzRo64tvHxB1IIV1U7Bgfy0CFq4Ln2jQ2Br2BkEVLAzid7wK/l3nN2OBcisebjC
/qkOBr/Ix31tzGOhFH4J+RXYxdlUfxVdzkEi6gSPOf1waHQD6jDPmqT723olCQav
qzxSZ+johaci+8/1j69Nj9gLpTB0FAjKyL7LdVSdDQjreZd0TNHk6u+1jzqBnDH+
7JDQc+QC0hgOstksHdzBbbAEOFdKIpu9Gp5TLAJusQCKtnZepOboWAUU/zdx43Zq
J4xGVKElvgIzYd7q1TLbrrIC3Z/VjbWNxOVQyuH4K7EKQ5WFHgszCaLJJzyTQbVL
TIU=
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIC4zCCAcugAwIBAgIJAKruJ7ZPXoA6MA0GCSqGSIb3DQEBBQUAMBkxFzAVBgNV
BAMMDlJvb3QgU2VydmVyIENBMB4XDTIwMDMwMzE2NTczMloXDTIwMDQwMjE2NTcz
MlowITEfMB0GA1UEAwwWSW50ZXJtZWRpYXRlIFNlcnZlciBDQTCCASIwDQYJKoZI
hvcNAQEBBQADggEPADCCAQoCggEBAMOqPllveHEI/lX5p/DBxhgb4nYVMb/JJjYZ
w+jkLcmkDfmgvwEersnVEpWvMGHbYu8zBKZ2yVMnuxxpn1nRSso4D7qrGi8XjgCS
LvTkSPaNUy/ixZetBnhZ6hlS+P1MCM9xCekFa7F9Ho5Va5Q5BD/AivyBL0ryUj9i
1f82shKVDYqYle8KL632MaYcrk2HHG990Gej+a9yQ817xmUmEQvFK2VBZvAI0SeI
f95XB8hLHua+BAlGw7wuQMQ6S4n4Ej7IxEbnKj1UqiSGoPWaUjALcLqdiHketABm
2LjmSFYwAfCyhDbM+7Fc1yaCx1cIzt/CCzStYqVlweDiQeGyx+UCAwEAAaMmMCQw
EgYDVR0TAQH/BAgwBgEB/wIBADAOBgNVHQ8BAf8EBAMCAYYwDQYJKoZIhvcNAQEF
BQADggEBANkB/SZNmV3FRpjpusBzYE36hx8GtjeHmQDlFQu2mGORmV6C/TXwhSR4
KJu2E12+M9oOEpGyURGQKUoV/BEE07ELm5MONSySvbxX63dx7KBeOIKtgQye12mD
JIU3QvT93QLccVQmJ7M7u+6K+TxJq0ZETOWbuOtwBX+8Ej3dO8VpAIjNtFBITBeZ
wFAzfZ9zt+zymxOEG9Ck96ieBr01KV1P7PLYzXI/nW91+LYn+041CeuoKIgU6mbP
bWqy+RorSEpPm9NAx27Mfp6p7yIzlWsZfcYb3KfLn8ntnvweczMcJ/vxBiE9NbBA
IaRJ0QQcxq3RH8oJHwcDp5mi1yxZ+vg=
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIICrjCCAZYCCQDatU6YWlNUpzANBgkqhkiG9w0BAQsFADAZMRcwFQYDVQQDDA5S
b290IFNlcnZlciBDQTAeFw0yMDAzMDMxNjU3MzJaFw0yMDA0MDIxNjU3MzJaMBkx
FzAVBgNVBAMMDlJvb3QgU2VydmVyIENBMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A
MIIBCgKCAQEA2wuwA9+ZhJtY+yXb4E1gYSR9g22F2esGUEZRGTCkuSK6fIvqWVJy
2ljmOvixe41QCaApQOtPqeShQGuzZ6vN5pFZUJiQU9eqoJ3IX/hHYbwMozYdeOv5
DrfcVne3VSH/dDQwBsMFk2lKYM0Zu0OhKvZiEgX81L2qSruhhXYZ5iTJAKd5ESQt
DkS/0rhi/C8jQS1Qty7QmFmKLc2vBIivKWw9m+0BJgbQxtUaRr/XNBJJbXbT8qGO
jsMi531ABvvgp5YKA2xbH2i1JDzoqmNmVAvXSyhj3+SrrH5FROpIctO2TuG4M4Tv
6IbDFvUpgZ+5aOCrougWHmnaaGbSqUiUUwIDAQABMA0GCSqGSIb3DQEBCwUAA4IB
AQCTu+oulRxnXm9Gp/mmxjEmNy+w/5dq03V89P5BiSYBYjdQD03K/COOJILMmNFR
usxJB3d5i75JGRCDE1spZZUefqp4M9NvX8mT60bxkeCQ78Dnf6S/dqBgoDA9846a
TOtk1oIXKXdr6HunxXE8OMrbODTEhjx37JtkoQqcZEGvaJhpjUWJKAdfChDyoony
GjbJv+YSl55hbn+rD/brMtauB38eRKeLS+284Fm/iktLiIsZxdPsU8NCRyXf9rj8
2wki5i3Jn9h4mcFmyRzoHKAd5Me36s/0ONQR+3BmJr0Cnxur1Dc01yJVxGm0hqQX
iM4kTpR+AV1zLVhEF8Jdjmvi
-----END CERTIFICATE-----
`
)

var (
	verificationTime = timeutil.MustParse(time.RFC3339, "2020-03-04T00:00:00Z")

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
