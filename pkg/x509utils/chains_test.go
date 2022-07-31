package x509utils

import (
	"crypto/x509"
	"testing"
	"time"

	"github.com/cloudflare/cfssl/helpers"
	"github.com/stackrox/rox/pkg/timeutil"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	pemChain = `
-----BEGIN CERTIFICATE-----
MIIEVTCCAz2gAwIBAgICEAAwDQYJKoZIhvcNAQELBQAwUTELMAkGA1UEBhMCVVMx
DzANBgNVBAgMBk9yZWdvbjEQMA4GA1UECgwHUmVkIEhhdDEfMB0GA1UEAwwWSW50
ZXJtZWRpYXRlIFNlcnZlciBDQTAeFw0yMjA3MjYxMDExMDhaFw0yMzA4MDUxMDEx
MDhaMGMxCzAJBgNVBAYTAlVTMRAwDgYDVQQIDAdFbmdsYW5kMREwDwYDVQQKDAhT
dGFja3JveDEvMC0GA1UEAwwmY3VzdG9tLXRscy1jZXJ0LmNlbnRyYWwuc3RhY2ty
b3gubG9jYWwwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDUuYn3X2pq
x6g4ykjARspcC/z8c4O43qIkjIhRIP4Y/GZ2JG/HWP+FnoIdeDQ/yct1HoD4GPTR
nc5XiGzaxm2VggaDENcRu8El505PPRb8W/uTVdZ+6cybFbnXZwUW5nmAVyXP/YFs
qr37XC7qHdbwV6Q5nP/Hsrlf6hsqoHT1ycpKWs9XCgyXt9bHPledK1I3c1Kn6/9o
/ZUALstiqpv6B3UovPMr3A+/Fg9WzRW8U7FZBrnHZPkiXT0BuLeF/vvjTCv62ATh
Z6ui79woooa249KYsz3yTC2TS/HQD5zmTayrk9znAPAQ+1zGUhBHU5PhLj1CPB8Y
O5YzViuSO+B/AgMBAAGjggEjMIIBHzAJBgNVHRMEAjAAMBEGCWCGSAGG+EIBAQQE
AwIGQDAzBglghkgBhvhCAQ0EJhYkT3BlblNTTCBHZW5lcmF0ZWQgU2VydmVyIENl
cnRpZmljYXRlMB0GA1UdDgQWBBQLWPfUAUPNYIAFc6ckdFdcLnWW7DCBhQYDVR0j
BH4wfIAUtPtFPhqwnVS/mixold2AvI9gBjmhYKReMFwxFzAVBgNVBAMMDlJvb3Qg
U2VydmVyIENBMRAwDgYDVQQKDAdSZWQgSGF0MQ8wDQYDVQQIDAZPcmVnb24xETAP
BgNVBAcMCFBvcnRsYW5kMQswCQYDVQQGEwJVU4ICEAAwDgYDVR0PAQH/BAQDAgWg
MBMGA1UdJQQMMAoGCCsGAQUFBwMBMA0GCSqGSIb3DQEBCwUAA4IBAQBNnDq8+6G7
M8Te7gdn5hd5/YYv3oJReQq0sJjwhUjaoPqMqDT5KqJ6dKT4hsMd11W53tC/q6VL
3WRd2KikIIHxHZLO04vAHqzcROnYu8Uj71RGAnhSskNA5+OWVZ5lR3bZ6A4YODYY
ACwywgS5GAeC0mtC8QeHME6Y7ahDaD0bCJx4MXqDHYijnLO9y1D29vtebWWyIdya
9074OiIoQp1adV3SAWyk6wM8R1p1X0+yRqL+837QjUKvlXJugAOdynHxfPP7yGiO
K/LOKBJT6T4CIjjNYSBj9qATDxYyg6mMqVr4CU4xe2xWv7NcYUWLC40jrBws+JBK
aUfnmq2yfE+6
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIDjzCCAnegAwIBAgICEAAwDQYJKoZIhvcNAQELBQAwXDEXMBUGA1UEAwwOUm9v
dCBTZXJ2ZXIgQ0ExEDAOBgNVBAoMB1JlZCBIYXQxDzANBgNVBAgMBk9yZWdvbjER
MA8GA1UEBwwIUG9ydGxhbmQxCzAJBgNVBAYTAlVTMB4XDTIyMDcyNjEwMDYzMloX
DTMyMDcyMzEwMDYzMlowUTELMAkGA1UEBhMCVVMxDzANBgNVBAgMBk9yZWdvbjEQ
MA4GA1UECgwHUmVkIEhhdDEfMB0GA1UEAwwWSW50ZXJtZWRpYXRlIFNlcnZlciBD
QTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBALf10aUQGoshuGiLyPcF
59voR3mcVWb3bCKeezEyArqrFOpe+cZG7bC6tLMUuaiK4BGTV7E9jZkcFN42qGfC
+1uOwiSU/jkudHs0iUqlgz+bsXScT1i6qxDnQVrEhBeA+dsr70NfJOYm51gus2hp
SWis/X+2FGCwlKKCpd2yA5epE0weBJh8XFPEoMk29ASWNJSDLMSqn+kMFHovmS98
t69IlITKzmLLHJ77p3w6675xS4tfgvUwdzxswBqRiV7T7YGL9wzVcFN/wyJ0HR2H
oAKz6E5est2AgseLC7vwwpDH9GXyzNjgSEPeqyVtlnd0HlgmUdffHv0kmtqZmHHY
idsCAwEAAaNmMGQwHQYDVR0OBBYEFLT7RT4asJ1Uv5osaJXdgLyPYAY5MB8GA1Ud
IwQYMBaAFFWi68XKfayYZvVP+FjudRWcUbHFMBIGA1UdEwEB/wQIMAYBAf8CAQAw
DgYDVR0PAQH/BAQDAgGGMA0GCSqGSIb3DQEBCwUAA4IBAQABNMYS/PL3qV+FIw5o
avQKLTdkGODJABLxOVdZu1zfHqjBujgckMx2Q5JCOhGKPTuHPksm/Nrq7ApoqWvP
VJUG6M1upSqQG+MABVp5ZtW3ajrUfcs9na5w73BQKHYybFLll7zNS8ndkv+ySYYU
SDvJd9n2Z+RpcAJUpRu3durELbx+hVjDY3b/dlgrUALJiWvuHkildGKFVm/jQbf1
j1fE0QJUTr4m3H5TIUudvJb0FnXLgRNVX5G5XQIJ1QudMje9OHxt90U/aLleGskh
JCuOVrKDqSwfcJevW4DKDrvvP8IV4D+KNRIUrIwZcz8qug6bLiZDeSZkdypgREgs
Klrj
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIDqTCCApGgAwIBAgIUWRwsQ+iZ0JF+KFFMnvJxVF2Wx7EwDQYJKoZIhvcNAQEL
BQAwXDEXMBUGA1UEAwwOUm9vdCBTZXJ2ZXIgQ0ExEDAOBgNVBAoMB1JlZCBIYXQx
DzANBgNVBAgMBk9yZWdvbjERMA8GA1UEBwwIUG9ydGxhbmQxCzAJBgNVBAYTAlVT
MB4XDTIyMDcyNjEwMDYyN1oXDTQyMDcyMTEwMDYyN1owXDEXMBUGA1UEAwwOUm9v
dCBTZXJ2ZXIgQ0ExEDAOBgNVBAoMB1JlZCBIYXQxDzANBgNVBAgMBk9yZWdvbjER
MA8GA1UEBwwIUG9ydGxhbmQxCzAJBgNVBAYTAlVTMIIBIjANBgkqhkiG9w0BAQEF
AAOCAQ8AMIIBCgKCAQEAokK4o+L/9UEKWuf3mZXL+sOftbM286IarBFHJhkszLw4
r9446+Su5m/FS6h+ipUt0HdFG/Ndx8oekqo8Et9/jm7F4px1JtK7lh9YkXQvNPZS
sdByToQpvRNPaTBO7ZwC0RwaCOkPN7ahqQeGw50e7D8HO83tfg7uLSeiOG8HNv4p
HZRdGnjcwEpN2C+DbtBNwWSj71PuxPnmstGnED+JPm3tJk24cIGcqLFjEdSvtjgg
Bv+KqSBQECRAFIx4rWNDoFVQ4LAoNHSdNglRd9KkytXulvs3JKYYsVkr4UZ/cEjM
LRfP4NobPwFpufBhuyYHn2Aw4U4WaUZXIPRT9fFV2QIDAQABo2MwYTAdBgNVHQ4E
FgQUVaLrxcp9rJhm9U/4WO51FZxRscUwHwYDVR0jBBgwFoAUVaLrxcp9rJhm9U/4
WO51FZxRscUwDwYDVR0TAQH/BAUwAwEB/zAOBgNVHQ8BAf8EBAMCAYYwDQYJKoZI
hvcNAQELBQADggEBAESrL5sHuDhasONrlMMNveF5l8RS6PkQDDnb7fVUDGlahrPQ
jCciU4ddl/EkjhnQDYIR50QKdKWpEg+q8Br1mVGIsu772Km8WN8CSDRf8mEmAjPz
y8YGwhUgKXArzNdHW9gdH5MIp5TivEKEedHF41EjxfoiU7bz9gBDEjaUQWK6vQR+
js3j9pFlXtzN7ypW19hmj/dEZQqhGTdQ+okN+ApX7n78eMd90CprlTqOs63QT7Kp
7HxliHliU8XIIhnRQS1in24Q00WuRYZtMa3MR3lgUHQWMQ8T3CcmQJlLgTr2cSU3
WCDhpxROiIEUjVRF/H58XvklMBNtpHxYk9cJk20=
-----END CERTIFICATE-----
`
)

var (
	verificationTime = timeutil.MustParse(time.RFC3339, "2022-07-27T00:00:00Z")

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
