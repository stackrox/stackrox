package m2m

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"mime"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/httputil/proxy"
)

type kubeServiceAccountVerifier struct {
	verifier *oidc.IDTokenVerifier
}

func (v *kubeServiceAccountVerifier) VerifyIDToken(ctx context.Context, rawIDToken string) (*oidc.IDToken, error) {
	return v.verifier.Verify(ctx, rawIDToken)
}

// JWKS represents a JSON Web Key Set
type jwks struct {
	Keys []jsonWebKey `json:"keys"`
}

// JSONWebKey represents a single JSON Web Key
type jsonWebKey struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type providerJSON struct {
	Issuer      string   `json:"issuer"`
	JWKSURL     string   `json:"jwks_uri"`
	UserInfoURL string   `json:"userinfo_endpoint"`
	Algorithms  []string `json:"id_token_signing_alg_values_supported"`
}

func NewKubeServiceAccountVerifier(ctx context.Context, issuer string, tlsConfig *tls.Config) (tokenVerifier, error) {
	token, err := readServiceAccountToken()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read kube service account token")
	}

	wellKnown := strings.TrimSuffix(issuer, "/") + "/.well-known/openid-configuration"
	client := &http.Client{
		Timeout:   time.Minute,
		Transport: proxy.RoundTripper(proxy.WithTLSConfig(tlsConfig)),
	}
	req, err := http.NewRequest("GET", wellKnown, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", resp.Status, body)
	}

	var p providerJSON
	err = unmarshalResp(resp, body, &p)
	if err != nil {
		return nil, fmt.Errorf("oidc: failed to decode provider discovery object: %w", err)
	}

	if p.Issuer != issuer {
		return nil, fmt.Errorf("oidc: issuer did not match the issuer returned by provider, expected %q got %q", issuer, p.Issuer)
	}

	keySet, err := newKeySet(ctx, client, p.JWKSURL)
	if err != nil {
		return nil, fmt.Errorf("oidc: failed to create KeySet: %w", err)
	}

	return &kubeServiceAccountVerifier{
		verifier: oidc.NewVerifier(issuer, keySet, &oidc.Config{ClientID: issuer}),
	}, nil
}

func newKeySet(ctx context.Context, client *http.Client, jwksURL string) (oidc.KeySet, error) {
	token, err := readServiceAccountToken()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read kube service account token")
	}

	req, err := http.NewRequest("GET", jwksURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", resp.Status, body)
	}

	var j jwks
	err = unmarshalResp(resp, body, &j)
	if err != nil {
		return nil, fmt.Errorf("oidc: failed to decode JWKS: %w", err)
	}
	var publicKeys []crypto.PublicKey
	for _, key := range j.Keys {
		if key.Kty == "RSA" {
			n, err := base64Decode(key.N)
			if err != nil {
				return nil, fmt.Errorf("failed to decode RSA public key modulus: %w", err)
			}
			e, err := base64Decode(key.E)
			if err != nil {
				return nil, fmt.Errorf("failed to decode RSA public key exponent: %w", err)
			}

			pubKey := &rsa.PublicKey{
				N: new(big.Int).SetBytes(n),
				E: int(new(big.Int).SetBytes(e).Int64()),
			}

			publicKeys = append(publicKeys, pubKey)
		} else {
			return nil, fmt.Errorf("unsupported key type: %s", key.Kty)
		}
	}

	return &oidc.StaticKeySet{PublicKeys: publicKeys}, nil
}

// Helper function to base64 decode a string
func base64Decode(input string) ([]byte, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(input)
	if err != nil {
		return nil, fmt.Errorf("base64 decode error: %w", err)
	}
	return decoded, nil
}

func unmarshalResp(r *http.Response, body []byte, v interface{}) error {
	err := json.Unmarshal(body, &v)
	if err == nil {
		return nil
	}
	ct := r.Header.Get("Content-Type")
	mediaType, _, parseErr := mime.ParseMediaType(ct)
	if parseErr == nil && mediaType == "application/json" {
		return fmt.Errorf("got Content-Type = application/json, but could not unmarshal as JSON: %w", err)
	}
	return fmt.Errorf("expected Content-Type = application/json, got %q: %w", ct, err)
}

func readServiceAccountToken() (string, error) {
	file, err := os.Open("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return "", fmt.Errorf("error opening service account token file: %v", err)
	}
	defer file.Close()

	token, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("error reading service account token file: %v", err)
	}

	return string(token), nil
}
