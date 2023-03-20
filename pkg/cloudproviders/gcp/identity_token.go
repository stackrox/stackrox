package gcp

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/cryptoutils"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/utils"
	"gopkg.in/square/go-jose.v2/jwt"
)

const (
	audienceBaseURI = `https://cloud-metadata.stackrox.io/gcp`
	baseIdentityURL = `http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/identity?format=full`

	nonceLen = 10
)

type identityTokenClaims struct {
	jwt.Claims

	Google struct {
		ComputeEngine struct {
			ProjectID string `json:"project_id"`
			Zone      string `json:"zone"`
		} `json:"compute_engine"`
	} `json:"google"`
}

func getIdentityToken(ctx context.Context, audience string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, baseIdentityURL, nil)
	if err != nil {
		return "", utils.ShouldErr(err)
	}
	req = req.WithContext(ctx)
	q := req.URL.Query()
	log.Errorf("[GCP getIdentityToken] request url %+q", req.URL)
	q.Add("audience", audience)
	log.Errorf("[GCP getIdentityToken] request aud %+q", audience)
	req.URL.RawQuery = q.Encode()
	log.Errorf("[GCP getIdentityToken] request rawquery %+q", req.URL.RawQuery)

	req.Header.Add("Metadata-Flavor", "Google")

	resp, err := metadataHTTPClient.Do(req)
	// Assume the service is unavailable if we encounter a transport error or a non-2xx status code
	if err != nil {
		log.Errorf("[GCP getIdentityToken] err from request %+q", err)
		return "", nil
	}
	defer utils.IgnoreError(resp.Body.Close)

	log.Errorf("[GCP getIdentityToken] req status code %+q", resp.StatusCode)
	if !httputil.Is2xxStatusCode(resp.StatusCode) {
		return "", nil
	}

	tokenBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("[GCP getIdentityToken] err reading response body %+q", err)
		return "", errors.Wrap(err, "reading response body")
	}

	return string(tokenBytes), nil
}

func getMetadataFromIdentityToken(ctx context.Context) (*gcpMetadata, error) {
	nonce, err := cryptoutils.NewNonceGenerator(nonceLen, rand.Reader).Nonce()
	if err != nil {
		log.Errorf("[GCP getMetadataFromIdentityToken] err generating nonce %+q", err)
		return nil, errors.Wrap(err, "generating nonce")
	}

	audience := fmt.Sprintf("%s#nonce=%s", audienceBaseURI, nonce)

	// Fetch Google's OAuth2 certs before and after retrieving the identity token to make sure we don't miss a
	// cert due to rotation.
	var certs certSet
	if err := certs.Fetch(ctx); err != nil {
		log.Errorf("Failed to fetch Google OAuth2 certs: %v", err)
	}

	identityToken, err := getIdentityToken(ctx, audience)
	if err != nil {
		log.Errorf("[GCP getMetadataFromIdentityToken] err getting identity token %+q", err)
		return nil, err
	}
	if identityToken == "" {
		log.Error("[GCP getMetadataFromIdentityToken] identity token is nil")
		return nil, nil
	}

	if err := certs.Fetch(ctx); err != nil {
		log.Errorf("Failed to fetch Google OAuth2 certs: %v", err)
	}

	parsedToken, err := jwt.ParseSigned(identityToken)
	if err != nil {
		log.Errorf("[GCP getMetadataFromIdentityToken] err generating jwt %+q", err)
		return nil, err
	}

	if len(parsedToken.Headers) != 1 {
		log.Errorf("[GCP getMetadataFromIdentityToken] parsed JWT should have exactly one header, has %+q", parsedToken.Headers)
		return nil, errors.Errorf("parsed JWT should have exactly one header, has %d", len(parsedToken.Headers))
	}

	kid := parsedToken.Headers[0].KeyID
	key := certs.GetKey(kid)
	if key == nil {
		log.Errorf("[GCP getMetadataFromIdentityToken] parsed JWT header referenced unknown key %+q", kid)
		return nil, errors.Errorf("parsed JWT header referenced unknown key %q", kid)
	}

	var claims identityTokenClaims

	if err := parsedToken.Claims(key, &claims); err != nil {
		log.Errorf("[GCP getMetadataFromIdentityToken] err retrieving claims %+q", err)
		return nil, errors.Wrap(err, "retrieving claims")
	}

	expectedClaims := jwt.Expected{
		Issuer:   "https://accounts.google.com",
		Audience: jwt.Audience{audience},
	}

	if err := claims.Validate(expectedClaims); err != nil {
		log.Errorf("[GCP getMetadataFromIdentityToken] err validating claims %+q", err)
		return nil, errors.Wrap(err, "validating claims")
	}

	if claims.Google.ComputeEngine.Zone == "" || claims.Google.ComputeEngine.ProjectID == "" {
		log.Error("[GCP getMetadataFromIdentityToken] identity token is missing required fields")
		return nil, errors.New("identity token is missing required fields")
	}

	return &gcpMetadata{
		Zone:      claims.Google.ComputeEngine.Zone,
		ProjectID: claims.Google.ComputeEngine.ProjectID,
	}, nil
}
