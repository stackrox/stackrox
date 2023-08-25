package tests

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	mathRand "math/rand"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/cloudflare/cfssl/helpers"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders/userpki"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/cryptoutils"
	"github.com/stackrox/rox/pkg/grpc/client/authn/tokenbased"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// generateCert generates a certificate with the given parent. If the parent is nil, a root cert is created.
func generateCert(t *testing.T, parent *x509.Certificate, signer crypto.Signer, isCA bool) (*x509.Certificate, crypto.Signer) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	serial, err := mtls.RandomSerial()
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber:          serial,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		BasicConstraintsValid: true,
		IsCA:                  isCA,
	}

	if parent == nil {
		// Assertions on the test.
		require.True(t, isCA, "Don't create a root cert that's not a CA")
		require.Nil(t, signer, "No parent cert passed, but signer was")
		parent = template
		signer = priv
	} else {
		require.NotNil(t, signer, "Parent cert was passed with no signer")
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, template, parent, priv.Public(), signer)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(certBytes)
	require.NoError(t, err)
	return cert, priv
}

func getTokenForUserPKIAuthProvider(t *testing.T, authProviderID string, tlsConf *tls.Config) string {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://%s/sso/providers/userpki/%s/authenticate", centralgrpc.RoxAPIEndpoint(t), authProviderID), nil)
	require.NoError(t, err)

	httpClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout:   10 * time.Second,
		Transport: &http.Transport{TLSClientConfig: tlsConf},
	}
	tokenResp, err := httpClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusSeeOther, tokenResp.StatusCode)
	loc, err := tokenResp.Location()
	require.NoError(t, err)
	// The fragment is in the form of a URL query string.
	parsedFragment, err := url.ParseQuery(loc.Fragment)
	require.NoError(t, err, "Failed to parse fragment from resp %+v", tokenResp)
	token := parsedFragment.Get("token")
	require.NotEmpty(t, token, "Got no token from resp %+v", tokenResp)
	return token
}

func validateAuthStatusResponseForClientCert(t *testing.T, cert *x509.Certificate, authStatus *v1.AuthStatus) {
	assert.Equal(t, "userpki", authStatus.GetAuthProvider().GetType())
	fingerprint := cryptoutils.CertFingerprint(cert)

	userIDAttributeIdx := sliceutils.FindMatching(authStatus.UserAttributes, func(attr *v1.UserAttribute) bool {
		return attr.Key == "userid"
	})
	assert.True(t, userIDAttributeIdx >= 0, "couldn't find userid attribute in resp %+v", authStatus)
	userIDAttr := authStatus.UserAttributes[userIDAttributeIdx]
	require.Len(t, userIDAttr.Values, 1, "unexpected number of values for userid attr in resp %+v", authStatus)
	assert.Equal(t, fmt.Sprintf("userpki:%s", fingerprint), userIDAttr.Values[0])
}

func getAuthStatus(t *testing.T, tlsConf *tls.Config, token string) (*v1.AuthStatus, error) {
	var opts []grpc.DialOption
	if token != "" {
		opts = append(opts, grpc.WithPerRPCCredentials(tokenbased.PerRPCCredentials(token)))
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	conn, err := clientconn.DialTLS(ctx, centralgrpc.RoxAPIEndpoint(t), tlsConf, opts...)
	require.NoError(t, err)
	client := v1.NewAuthServiceClient(conn)
	return client.GetAuthStatus(ctx, &v1.Empty{})
}

func tlsConfWithCertChain(leafKey crypto.PrivateKey, leafCert *x509.Certificate, otherCerts ...*x509.Certificate) *tls.Config {
	certChain := make([][]byte, 0, len(otherCerts)+1)
	certChain = append(certChain, leafCert.Raw)
	for _, cert := range otherCerts {
		certChain = append(certChain, cert.Raw)
	}

	return &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{{Certificate: certChain, PrivateKey: leafKey}},
	}
}

// This test validates that client certs work correctly in cases that result in multiple
// VerifiedChains being constructed by the client certs.
// This happens because we have a cert chain like this:
// rootCA->secondCA->leaf
// and we add _both_ rootCA and secondCA as valid CAs in the created auth provider.
// The client presents the chain secondCA->leaf.
// Therefore, VerifiedChains will be [leaf secondCA] and [leaf secondCA rootCA].
func TestClientCAAuthWithMultipleVerifiedChains(t *testing.T) {
	rootCA, rootCAKey := generateCert(t, nil, nil, true)
	secondCA, secondCAKey := generateCert(t, rootCA, rootCAKey, true)
	leafCert, leafKey := generateCert(t, secondCA, secondCAKey, false)
	secondLeafCert, secondLeafKey := generateCert(t, secondCA, secondCAKey, false)

	req := &v1.PostAuthProviderRequest{
		Provider: &storage.AuthProvider{
			Type:    userpki.TypeName,
			Name:    fmt.Sprintf("test-%d", mathRand.Int()),
			Enabled: true,
			Config: map[string]string{
				userpki.ConfigKeys: string(helpers.EncodeCertificatesPEM([]*x509.Certificate{rootCA, secondCA})),
			},
		},
	}
	conn := centralgrpc.GRPCConnectionToCentral(t)
	authService := v1.NewAuthProviderServiceClient(conn)
	groupService := v1.NewGroupServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	createdAuthProvider, err := authService.PostAuthProvider(ctx, req)
	require.NoError(t, err)
	_, err = groupService.CreateGroup(ctx, &storage.Group{
		Props: &storage.GroupProperties{
			AuthProviderId: createdAuthProvider.GetId(),
		},
		RoleName: "Admin",
	})
	require.NoError(t, err)

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		_, err := authService.DeleteAuthProvider(ctx, &v1.DeleteByIDWithForce{Id: createdAuthProvider.GetId()})
		require.NoError(t, err)
	}()

	// Test direct access using only client certs.
	tlsConfWithLeaf := tlsConfWithCertChain(leafKey, leafCert, secondCA)
	authStatus, err := getAuthStatus(t, tlsConfWithLeaf, "")
	require.NoError(t, err)
	validateAuthStatusResponseForClientCert(t, leafCert, authStatus)
	assert.Empty(t, cmp.Diff(createdAuthProvider, authStatus.GetAuthProvider(), cmpopts.IgnoreFields(storage.AuthProvider{}, "Config")))

	// Simulate the flow used in the browser, where the certs are exchanged for a token.
	token := getTokenForUserPKIAuthProvider(t, createdAuthProvider.GetId(), tlsConfWithLeaf)

	// If only token is passed but with no client certs, we expect an error.
	_, err = getAuthStatus(t, &tls.Config{InsecureSkipVerify: true}, token)
	assert.Error(t, err)

	// Token plus other, non-matching cert => should error.
	_, err = getAuthStatus(t, tlsConfWithCertChain(secondLeafKey, secondLeafCert, secondCA), token)
	assert.Error(t, err)

	// Token plus matching cert => things should work.
	authStatusWithToken, err := getAuthStatus(t, tlsConfWithLeaf, token)
	require.NoError(t, err)
	assert.Empty(t, cmp.Diff(createdAuthProvider, authStatusWithToken.GetAuthProvider(),
		cmpopts.IgnoreFields(storage.AuthProvider{}, "Config", "Validated", "Active", "LastUpdated")))
	validateAuthStatusResponseForClientCert(t, leafCert, authStatusWithToken)
}

func TestClientCARequested(t *testing.T) {
	t.Parallel()

	clientCAFile := os.Getenv("CLIENT_CA_PATH")
	require.NotEmpty(t, clientCAFile, "no client CA file path set")
	pemBytes, err := os.ReadFile(clientCAFile)
	require.NoErrorf(t, err, "Could not read client CA file %s", clientCAFile)
	caCert, err := helpers.ParseCertificatePEM(pemBytes)
	require.NoError(t, err, "Could not parse client CA PEM data")

	var acceptableCAs [][]byte
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         "central.stackrox",
		GetClientCertificate: func(cri *tls.CertificateRequestInfo) (*tls.Certificate, error) {
			acceptableCAs = cri.AcceptableCAs
			return &tls.Certificate{}, nil
		},
	}

	conn, err := tls.Dial("tcp", centralgrpc.RoxAPIEndpoint(t), tlsConf)
	require.NoError(t, err, "could not connect to central")
	_ = conn.Handshake()
	_ = conn.Close()

	found := false
	for _, acceptableCA := range acceptableCAs {
		if bytes.Equal(acceptableCA, caCert.RawSubject) {
			found = true
			break
		}
	}

	assert.True(t, found, "server did not request appropriate client certs")
}
