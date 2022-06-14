package aws

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/cloudflare/cfssl/helpers"
	"github.com/fullsailor/pkcs7"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	pkcs7URL = `http://169.254.169.254/latest/dynamic/instance-identity/pkcs7`
)

var (
	awsCerts []*x509.Certificate
)

func init() {
	var err error
	awsCerts, err = helpers.ParseCertificatesPEM([]byte(`
-----BEGIN CERTIFICATE-----
MIIC7TCCAq0CCQCWukjZ5V4aZzAJBgcqhkjOOAQDMFwxCzAJBgNVBAYTAlVTMRkw
FwYDVQQIExBXYXNoaW5ndG9uIFN0YXRlMRAwDgYDVQQHEwdTZWF0dGxlMSAwHgYD
VQQKExdBbWF6b24gV2ViIFNlcnZpY2VzIExMQzAeFw0xMjAxMDUxMjU2MTJaFw0z
ODAxMDUxMjU2MTJaMFwxCzAJBgNVBAYTAlVTMRkwFwYDVQQIExBXYXNoaW5ndG9u
IFN0YXRlMRAwDgYDVQQHEwdTZWF0dGxlMSAwHgYDVQQKExdBbWF6b24gV2ViIFNl
cnZpY2VzIExMQzCCAbcwggEsBgcqhkjOOAQBMIIBHwKBgQCjkvcS2bb1VQ4yt/5e
ih5OO6kK/n1Lzllr7D8ZwtQP8fOEpp5E2ng+D6Ud1Z1gYipr58Kj3nssSNpI6bX3
VyIQzK7wLclnd/YozqNNmgIyZecN7EglK9ITHJLP+x8FtUpt3QbyYXJdmVMegN6P
hviYt5JH/nYl4hh3Pa1HJdskgQIVALVJ3ER11+Ko4tP6nwvHwh6+ERYRAoGBAI1j
k+tkqMVHuAFcvAGKocTgsjJem6/5qomzJuKDmbJNu9Qxw3rAotXau8Qe+MBcJl/U
hhy1KHVpCGl9fueQ2s6IL0CaO/buycU1CiYQk40KNHCcHfNiZbdlx1E9rpUp7bnF
lRa2v1ntMX3caRVDdbtPEWmdxSCYsYFDk4mZrOLBA4GEAAKBgEbmeve5f8LIE/Gf
MNmP9CM5eovQOGx5ho8WqD+aTebs+k2tn92BBPqeZqpWRa5P/+jrdKml1qx4llHW
MXrs3IgIb6+hUIB+S8dz8/mmO0bpr76RoZVCXYab2CZedFut7qc3WUH9+EUAH5mw
vSeDCOUMYQR7R9LINYwouHIziqQYMAkGByqGSM44BAMDLwAwLAIUWXBlk40xTwSw
7HX32MxXYruse9ACFBNGmdX2ZBrVNGrN9N2f6ROk0k9K
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIICuzCCAiQCCQDrSGnlRgvSazANBgkqhkiG9w0BAQUFADCBoTELMAkGA1UEBhMC
VVMxCzAJBgNVBAgTAldBMRAwDgYDVQQHEwdTZWF0dGxlMRMwEQYDVQQKEwpBbWF6
b24uY29tMRYwFAYDVQQLEw1FQzIgQXV0aG9yaXR5MRowGAYDVQQDExFFQzIgQU1J
IEF1dGhvcml0eTEqMCgGCSqGSIb3DQEJARYbZWMyLWluc3RhbmNlLWlpZEBhbWF6
b24uY29tMB4XDTExMDgxMjE3MTgwNVoXDTIxMDgwOTE3MTgwNVowgaExCzAJBgNV
BAYTAlVTMQswCQYDVQQIEwJXQTEQMA4GA1UEBxMHU2VhdHRsZTETMBEGA1UEChMK
QW1hem9uLmNvbTEWMBQGA1UECxMNRUMyIEF1dGhvcml0eTEaMBgGA1UEAxMRRUMy
IEFNSSBBdXRob3JpdHkxKjAoBgkqhkiG9w0BCQEWG2VjMi1pbnN0YW5jZS1paWRA
YW1hem9uLmNvbTCBnzANBgkqhkiG9w0BAQEFAAOBjQAwgYkCgYEAqaIcGFFTx/SO
1W5G91jHvyQdGP25n1Y91aXCuOOWAUTvSvNGpXrI4AXNrQF+CmIOC4beBASnHCx0
82jYudWBBl9Wiza0psYc9flrczSzVLMmN8w/c78F/95NfiQdnUQPpvgqcMeJo82c
gHkLR7XoFWgMrZJqrcUK0gnsQcb6kakCAwEAATANBgkqhkiG9w0BAQUFAAOBgQDF
VH0+UGZr1LCQ78PbBH0GreiDqMFfa+W8xASDYUZrMvY3kcIelkoIazvi4VtPO7Qc
yAiLr6nkk69Tr/MITnmmsZJZPetshqBndRyL+DaTRnF0/xvBQXj5tEh+AmRjvGtp
6iS1rQoNanN8oEcT2j4b48rmCmnDhRoBcFHwCYs/3w==
-----END CERTIFICATE-----
`))
	utils.CrashOnError(err)
}

func getIdentityDocFromPKCS7(ctx context.Context) (*ec2metadata.EC2InstanceIdentityDocument, error) {
	req, err := http.NewRequest(http.MethodGet, pkcs7URL, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	resp, err := httpClient.Do(req)
	// Assume the service is unavailable if we encounter a transport error or a non-2xx status code
	if err != nil {
		return nil, nil
	}
	defer utils.IgnoreError(resp.Body.Close)

	if !httputil.Is2xxStatusCode(resp.StatusCode) {
		return nil, nil
	}

	b64Bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	rawPKCS7, err := base64.StdEncoding.DecodeString(string(b64Bytes))
	if err != nil {
		return nil, err
	}

	pkcs7Data, err := pkcs7.Parse(rawPKCS7)
	if err != nil {
		return nil, err
	}

	pkcs7Data.Certificates = awsCerts
	if err := pkcs7Data.Verify(); err != nil {
		return nil, errors.Wrap(err, "verifying PKCS7 signature")
	}

	var instanceIDDoc ec2metadata.EC2InstanceIdentityDocument
	if err := json.Unmarshal(pkcs7Data.Content, &instanceIDDoc); err != nil {
		return nil, errors.Wrap(err, "unmarshaling instance identity document")
	}

	return &instanceIDDoc, nil
}
