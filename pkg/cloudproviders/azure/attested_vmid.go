package azure

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/fullsailor/pkcs7"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	attestedMetadataBaseURL = `http://169.254.169.254/metadata/attested/document?api-version=2018-10-01`

	timeFormat = `01/02/06 15:04:05 -0700`
)

var (
	allowedCNsRegExp = regexp.MustCompile(`^(.*\.)?metadata\.(azure\.(com|us|cn)|microsoftazure\.de)$`)
)

type attestedMetadataResponse struct {
	Encoding  string `json:"encoding"`
	Signature string `json:"signature"`
}

type attestedMetadata struct {
	TimeStamp struct {
		CreatedOn string `json:"createdOn"`
		ExpiresOn string `json:"expiresOn"`
	} `json:"timeStamp"`
	VMID string `json:"vmId"`
}

func getAttestedVMID(ctx context.Context) (string, error) {
	req, err := http.NewRequest(http.MethodGet, attestedMetadataBaseURL, nil)
	if err != nil {
		return "", utils.ShouldErr(err)
	}
	req = req.WithContext(ctx)

	// It would be nice to add the nonce here, but that doesn't work. It will just be ignored, even
	// if you just run the example `curl` command line from
	// https://docs.microsoft.com/en-us/azure/virtual-machines/windows/instance-metadata-service#attested-data .
	req.Header.Add("Metadata", "True")

	resp, err := metadataHTTPClient.Do(req)
	// Assume the service is unavailable if we encounter a transport error or a non-2xx status code
	if err != nil {
		return "", nil
	}

	defer utils.IgnoreError(resp.Body.Close)

	if !httputil.Is2xxStatusCode(resp.StatusCode) {
		return "", nil
	}

	var attestedMDResponse attestedMetadataResponse
	if err := json.NewDecoder(resp.Body).Decode(&attestedMDResponse); err != nil {
		return "", errors.Wrap(err, "unmarshaling response")
	}

	if attestedMDResponse.Encoding != "pkcs7" {
		return "", errors.Errorf("unsupported encoding %q, only pkcs7 is supported", attestedMDResponse.Encoding)
	}

	raw, err := base64.StdEncoding.DecodeString(attestedMDResponse.Signature)
	if err != nil {
		return "", errors.Wrap(err, "base64 decoding PKCS7 data")
	}

	pkcs7Data, err := pkcs7.Parse(raw)
	if err != nil {
		return "", errors.Wrap(err, "parsing PKCS7 data")
	}

	if err := pkcs7Data.Verify(); err != nil {
		return "", errors.Wrap(err, "verifying PKCS7 data")
	}

	signer := pkcs7Data.GetOnlySigner()
	if signer == nil {
		return "", errors.New("expected PKCS7 data to be signed by a single signer")
	}

	if !allowedCNsRegExp.MatchString(signer.Subject.CommonName) {
		return "", errors.Errorf("invalid CN %q of signer", signer.Subject.CommonName)
	}

	intermediateCertPool := x509.NewCertPool()
	for _, intermediateCertURL := range signer.IssuingCertificateURL {
		resp, err := certificateHTTPClient.Get(intermediateCertURL)
		if err != nil {
			return "", errors.Wrap(err, "retrieving intermediate CA certificate")
		}
		defer utils.IgnoreError(resp.Body.Close)

		bytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", errors.Wrap(err, "retrieving intermediate CA certificate")
		}

		cert, err := x509.ParseCertificate(bytes)
		if err != nil {
			return "", errors.Wrap(err, "parsing intermediate CA certificate")
		}
		intermediateCertPool.AddCert(cert)
	}

	verifyOpts := x509.VerifyOptions{
		Roots:         nil, // use system cert pool
		Intermediates: intermediateCertPool,
	}

	if _, err := signer.Verify(verifyOpts); err != nil {
		return "", errors.Wrap(err, "verifying signer cert")
	}

	var attestedMD attestedMetadata
	if err := json.Unmarshal(pkcs7Data.Content, &attestedMD); err != nil {
		return "", errors.Wrap(err, "unmarshalling attested metadata content")
	}

	createdOn, err := time.Parse(timeFormat, attestedMD.TimeStamp.CreatedOn)
	if err != nil {
		return "", errors.Wrap(err, "parsing `created on` timestamp")
	}
	if createdOn.After(time.Now()) {
		return "", errors.Errorf("`created on` timestamp %v is in the future", createdOn)
	}

	expiresOn, err := time.Parse(timeFormat, attestedMD.TimeStamp.ExpiresOn)
	if err != nil {
		return "", errors.Wrap(err, "parsing `expires on` timestamp")
	}
	if !expiresOn.After(time.Now()) {
		return "", errors.Errorf("attested metadata expired %v", expiresOn)
	}

	return attestedMD.VMID, nil
}
