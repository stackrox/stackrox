package aws

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/fullsailor/pkcs7"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	tokenURL = `http://169.254.169.254/latest/api/token`
	pkcs7URL = `http://169.254.169.254/latest/dynamic/instance-identity/pkcs7`
)

func getIdentityDocFromPKCS7(ctx context.Context) (*ec2metadata.EC2InstanceIdentityDocument, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, tokenURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("X-aws-ec2-metadata-token-ttl-seconds", "21600")

	resp, err := httpClient.Do(req)
	// Assume the service is unavailable if we encounter a transport error or a non-2xx status code
	if err != nil {
		return nil, nil
	}
	defer utils.IgnoreError(resp.Body.Close)

	if !httputil.Is2xxStatusCode(resp.StatusCode) {
		return nil, nil
	}

	token, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	req, err = http.NewRequestWithContext(ctx, http.MethodGet, pkcs7URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("X-aws-ec2-metadata-token", string(token))

	resp, err = httpClient.Do(req)
	// Assume the service is unavailable if we encounter a transport error or a non-2xx status code
	if err != nil {
		return nil, nil
	}
	defer utils.IgnoreError(resp.Body.Close)

	b64, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	rawPKCS7, err := base64.StdEncoding.DecodeString(string(b64))
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
