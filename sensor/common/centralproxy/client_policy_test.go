package centralproxy

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/centralproxy"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/pods"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestClientPolicyBootstrapAndLiveUpdates(t *testing.T) {
	t.Setenv("POD_NAMESPACE", "proxy-policy-test")
	ctx := context.Background()
	client := fake.NewClientset()
	policy := NewClientPolicy(client)
	request := httptest.NewRequest(http.MethodGet, "/v1/metadata", nil)
	request.Header.Set("Authorization", "Bearer reviewed-token")
	user := &authenticationv1.UserInfo{Username: "alice"}
	_, err := policy.Authorize(ctx, request, user)
	require.Error(t, err)
	require.NoError(t, policy.EnsureDefaults(ctx))
	cm, err := client.CoreV1().ConfigMaps(pods.GetPodNamespace()).Get(ctx, ClientPolicyConfigMapName, metav1.GetOptions{})
	require.NoError(t, err)
	var defaults clientPolicyDocument
	require.NoError(t, json.Unmarshal([]byte(cm.Data[ClientPolicyDataKey]), &defaults))
	require.Equal(t, clientPolicyVersion, defaults.Version)
	require.Len(t, defaults.Clients, 2)
	assert.Equal(t, env.SecureMetricsClientCertCN.Setting(), defaults.Clients[0].Subject)
	assert.Equal(t, centralproxy.MetricsPermissionSetID, defaults.Clients[0].PermissionSetID)
	assert.Equal(t, "*", defaults.Clients[1].Subject)
	assert.Equal(t, centralproxy.ConsoleProxyPermissionSetID, defaults.Clients[1].PermissionSetID)
	consoleScopes, err := tokens.HTTPRequestScopesFromProto(defaults.Clients[1].HTTPScopes)
	require.NoError(t, err)
	assert.Equal(t, tokens.HTTPRequestScopesByPath(centralproxy.ConsoleHTTPScopes), consoleScopes)
	grant, err := policy.Authorize(ctx, request, user)
	require.NoError(t, err)
	assert.Equal(t, clientAuthKubernetesBearer, grant.AuthType)
	assert.Equal(t, "alice", grant.Subject)
	assert.Equal(t, centralproxy.ConsoleProxyPermissionSetID, grant.PermissionSetID)
	assert.Equal(t, defaults.Clients[1].HTTPScopes, grant.HTTPRequestScopes)

	// A new exact user policy overrides the wildcard, without weakening other users.
	defaults.Clients = append(defaults.Clients, clientPolicyEntry{
		AuthType: clientAuthKubernetesBearer, Subject: "alice", PermissionSetID: "alice-permissions",
		HTTPScopes: consolePolicyHTTPScopes(),
	})
	setClientPolicyConfigMap(t, ctx, client, defaults)
	grant, err = policy.Authorize(ctx, request, user)
	require.NoError(t, err)
	assert.Equal(t, "alice-permissions", grant.PermissionSetID)
	grant, err = policy.Authorize(ctx, request, &authenticationv1.UserInfo{Username: "bob"})
	require.NoError(t, err)
	assert.Equal(t, centralproxy.ConsoleProxyPermissionSetID, grant.PermissionSetID)
	defaults.Version++
	setClientPolicyConfigMap(t, ctx, client, defaults)
	_, err = policy.Authorize(ctx, request, user)
	require.Error(t, err)

	// Removing an established policy never recreates it or falls back to defaults.
	require.NoError(t, client.CoreV1().ConfigMaps(pods.GetPodNamespace()).Delete(ctx, ClientPolicyConfigMapName, metav1.DeleteOptions{}))
	_, err = policy.Authorize(ctx, request, user)
	require.Error(t, err)
	require.NoError(t, policy.EnsureDefaults(ctx))
	_, err = client.CoreV1().ConfigMaps(pods.GetPodNamespace()).Get(ctx, ClientPolicyConfigMapName, metav1.GetOptions{})
	require.Error(t, err)
}

func TestClientPolicyRejectsUnreviewedOrOutOfScopeBearers(t *testing.T) {
	ctx := context.Background()
	client := fake.NewClientset()
	policy := NewClientPolicy(client)
	require.NoError(t, policy.EnsureDefaults(ctx))
	for name, tc := range map[string]struct {
		method, path string
		bearer       string
		user         *authenticationv1.UserInfo
	}{
		"no review":        {method: http.MethodGet, path: "/v1/metadata", bearer: "Bearer token"},
		"empty username":   {method: http.MethodGet, path: "/v1/metadata", bearer: "Bearer token", user: &authenticationv1.UserInfo{}},
		"no bearer header": {method: http.MethodGet, path: "/v1/metadata", user: &authenticationv1.UserInfo{Username: "alice"}},
		"wrong path":       {method: http.MethodGet, path: "/v1/secrets", bearer: "Bearer token", user: &authenticationv1.UserInfo{Username: "alice"}},
		"wrong method":     {method: http.MethodPost, path: "/v1/metadata", bearer: "Bearer token", user: &authenticationv1.UserInfo{Username: "alice"}},
		"encoded path":     {method: http.MethodGet, path: "/v1/%6detadata", bearer: "Bearer token", user: &authenticationv1.UserInfo{Username: "alice"}},
		"metrics path":     {method: http.MethodGet, path: "/metrics", bearer: "Bearer token", user: &authenticationv1.UserInfo{Username: "alice"}},
	} {
		t.Run(name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.bearer != "" {
				r.Header.Set("Authorization", tc.bearer)
			}
			_, err := policy.Authorize(ctx, r, tc.user)
			require.Error(t, err)
		})
	}
}

func TestClientPolicyEmptyEntriesDisableClients(t *testing.T) {
	ctx := context.Background()
	client := fake.NewClientset()
	policy := NewClientPolicy(client)
	require.NoError(t, policy.EnsureDefaults(ctx))
	setClientPolicyConfigMap(t, ctx, client, clientPolicyDocument{Version: clientPolicyVersion, Clients: []clientPolicyEntry{}})
	r := httptest.NewRequest(http.MethodGet, "/v1/metadata", nil)
	r.Header.Set("Authorization", "Bearer token")
	_, err := policy.Authorize(ctx, r, &authenticationv1.UserInfo{Username: "alice"})
	require.Error(t, err)
}

func TestClientPolicyMTLSMatchesOnlyReferencedCA(t *testing.T) {
	ctx := context.Background()
	ca, caKey, caPEM := clientPolicyTestCA(t)
	otherCA, otherCAKey, otherCAPEM := clientPolicyTestCA(t)
	ref := clientPolicyCARef{
		Namespace: env.SecureMetricsClientCANamespace.Setting(),
		ConfigMap: env.SecureMetricsClientCAConfigMap.Setting(),
		Key:       env.SecureMetricsClientCAKey.Setting(),
	}
	client := fake.NewClientset(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: ref.ConfigMap, Namespace: ref.Namespace},
		Data:       map[string]string{ref.Key: caPEM},
	})
	policy := NewClientPolicy(client)
	require.NoError(t, policy.EnsureDefaults(ctx))
	otherRef := clientPolicyCARef{Namespace: ref.Namespace, ConfigMap: "second-client-ca", Key: ref.Key}
	_, err := client.CoreV1().ConfigMaps(otherRef.Namespace).Create(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: otherRef.ConfigMap, Namespace: otherRef.Namespace},
		Data:       map[string]string{otherRef.Key: otherCAPEM},
	}, metav1.CreateOptions{})
	require.NoError(t, err)
	doc := defaultClientPolicy()
	otherRefJSON, err := json.Marshal(otherRef)
	require.NoError(t, err)
	doc.Clients = append(doc.Clients, clientPolicyEntry{
		AuthType: clientAuthMTLS, Subject: "second-client", CARef: otherRefJSON,
		PermissionSetID: "second-permissions",
		HTTPScopes:      doc.Clients[0].HTTPScopes,
	})
	setClientPolicyConfigMap(t, ctx, client, doc)
	good := signedTestCertificate(t, ca, caKey, env.SecureMetricsClientCertCN.Setting(), x509.ExtKeyUsageClientAuth)
	other := signedTestCertificate(t, otherCA, otherCAKey, env.SecureMetricsClientCertCN.Setting(), x509.ExtKeyUsageClientAuth)
	second := signedTestCertificate(t, otherCA, otherCAKey, "second-client", x509.ExtKeyUsageClientAuth)
	wrongCN := signedTestCertificate(t, ca, caKey, "other-client", x509.ExtKeyUsageClientAuth)
	wrongEKU := signedTestCertificate(t, ca, caKey, env.SecureMetricsClientCertCN.Setting(), x509.ExtKeyUsageServerAuth)
	expired := signedTestCertificate(t, ca, caKey, env.SecureMetricsClientCertCN.Setting(), x509.ExtKeyUsageClientAuth, time.Now().Add(-time.Minute))
	request := func(cert tls.Certificate) *http.Request {
		r := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		leaf, err := x509.ParseCertificate(cert.Certificate[0])
		require.NoError(t, err)
		r.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{leaf}}
		return r
	}
	grant, err := policy.Authorize(ctx, request(good), nil)
	require.NoError(t, err)
	assert.Equal(t, clientAuthMTLS, grant.AuthType)
	assert.Equal(t, env.SecureMetricsClientCertCN.Setting(), grant.Subject)
	assert.Equal(t, centralproxy.MetricsPermissionSetID, grant.PermissionSetID)
	grant, err = policy.Authorize(ctx, request(second), nil)
	require.NoError(t, err)
	assert.Equal(t, "second-permissions", grant.PermissionSetID)
	for name, cert := range map[string]tls.Certificate{
		"wrong CA": other, "wrong CN": wrongCN, "wrong EKU": wrongEKU, "expired": expired,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := policy.Authorize(ctx, request(cert), nil)
			require.Error(t, err)
		})
	}
	// A bearer token cannot be used as a fallback for an untrusted client certificate.
	bad := request(other)
	bad.Header.Set("Authorization", "Bearer reviewed-token")
	_, err = policy.Authorize(ctx, bad, &authenticationv1.UserInfo{Username: "alice"})
	require.Error(t, err)

	caConfigMap, err := client.CoreV1().ConfigMaps(ref.Namespace).Get(ctx, ref.ConfigMap, metav1.GetOptions{})
	require.NoError(t, err)
	caConfigMap.Data[ref.Key] = otherCAPEM
	_, err = client.CoreV1().ConfigMaps(ref.Namespace).Update(ctx, caConfigMap, metav1.UpdateOptions{})
	require.NoError(t, err)
	_, err = policy.Authorize(ctx, request(good), nil)
	require.Error(t, err)
	_, err = policy.Authorize(ctx, request(other), nil)
	require.NoError(t, err)
	require.NoError(t, client.CoreV1().ConfigMaps(ref.Namespace).Delete(ctx, ref.ConfigMap, metav1.DeleteOptions{}))
	_, err = policy.Authorize(ctx, request(other), nil)
	require.Error(t, err)
}

func TestClientPolicyRejectsMalformedPolicies(t *testing.T) {
	defaults, err := json.Marshal(defaultClientPolicy())
	require.NoError(t, err)
	for name, change := range map[string]func(*clientPolicyDocument){
		"unsupported version": func(p *clientPolicyDocument) { p.Version++ },
		"duplicate identity":  func(p *clientPolicyDocument) { p.Clients = append(p.Clients, p.Clients[0]) },
		"unknown auth type":   func(p *clientPolicyDocument) { p.Clients[0].AuthType = "token" },
		"missing CA":          func(p *clientPolicyDocument) { p.Clients[0].CARef = nil },
		"bearer CA":           func(p *clientPolicyDocument) { p.Clients[1].CARef = p.Clients[0].CARef },
		"empty scopes":        func(p *clientPolicyDocument) { p.Clients[0].HTTPScopes = nil },
		"empty methods":       func(p *clientPolicyDocument) { p.Clients[0].HTTPScopes[0].Methods = nil },
		"invalid method":      func(p *clientPolicyDocument) { p.Clients[0].HTTPScopes[0].Methods = []string{"Get"} },
		"noncanonical path":   func(p *clientPolicyDocument) { p.Clients[0].HTTPScopes[0].Path = "/%6detrics" },
		"duplicate scope": func(p *clientPolicyDocument) {
			p.Clients[0].HTTPScopes = append(p.Clients[0].HTTPScopes, p.Clients[0].HTTPScopes[0])
		},
	} {
		t.Run(name, func(t *testing.T) {
			var p clientPolicyDocument
			require.NoError(t, json.Unmarshal(defaults, &p))
			change(&p)
			data, err := json.Marshal(p)
			require.NoError(t, err)
			_, err = parseClientPolicy(&corev1.ConfigMap{Data: map[string]string{ClientPolicyDataKey: string(data)}})
			require.Error(t, err)
		})
	}
	for name, data := range map[string]string{
		"unknown top field":    string(defaults[:len(defaults)-1]) + `,"unrecognized":true}`,
		"unknown nested field": strings.Replace(string(defaults), `"authType":"mtls"`, `"authType":"mtls","unrecognized":true`, 1),
		"duplicate top key":    strings.Replace(string(defaults), `"version":1`, `"version":1,"version":1`, 1),
		"duplicate nested key": strings.Replace(string(defaults), `"authType":"mtls"`, `"authType":"mtls","authType":"mtls"`, 1),
		"trailing JSON":        string(defaults) + `{}`,
		"null document":        `null`,
		"missing data key":     "",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := parseClientPolicy(&corev1.ConfigMap{Data: map[string]string{ClientPolicyDataKey: data}})
			require.Error(t, err)
		})
	}
}

func TestClientPolicyDoesNotReplaceInvalidExistingConfigMap(t *testing.T) {
	t.Setenv("POD_NAMESPACE", "policy-invalid-test")
	ctx := context.Background()
	client := fake.NewClientset(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: ClientPolicyConfigMapName, Namespace: pods.GetPodNamespace()},
		Data:       map[string]string{ClientPolicyDataKey: `{"version":2,"clients":[]}`},
	})
	policy := NewClientPolicy(client)
	require.Error(t, policy.EnsureDefaults(ctx))
	cm, err := client.CoreV1().ConfigMaps(pods.GetPodNamespace()).Get(ctx, ClientPolicyConfigMapName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, `{"version":2,"clients":[]}`, cm.Data[ClientPolicyDataKey])
	_, err = policy.Authorize(ctx, httptest.NewRequest(http.MethodGet, "/metrics", nil), nil)
	require.Error(t, err)
}

func TestParseClientCARootsRejectsPartialOrInvalidBundles(t *testing.T) {
	_, _, caPEM := clientPolicyTestCA(t)
	for name, data := range map[string]string{
		"empty": "", "invalid PEM": "not a certificate", "trailing junk": caPEM + "junk",
		"noncertificate": strings.Replace(caPEM, "CERTIFICATE", "PRIVATE KEY", 2),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := parseClientCARoots([]byte(data))
			require.Error(t, err)
		})
	}
	_, err := parseClientCARoots([]byte(caPEM))
	require.NoError(t, err)
}

func setClientPolicyConfigMap(t *testing.T, ctx context.Context, client *fake.Clientset, doc clientPolicyDocument) {
	t.Helper()
	data, err := json.Marshal(doc)
	require.NoError(t, err)
	cm, err := client.CoreV1().ConfigMaps(pods.GetPodNamespace()).Get(ctx, ClientPolicyConfigMapName, metav1.GetOptions{})
	require.NoError(t, err)
	cm.Data[ClientPolicyDataKey] = string(data)
	_, err = client.CoreV1().ConfigMaps(pods.GetPodNamespace()).Update(ctx, cm, metav1.UpdateOptions{})
	require.NoError(t, err)
}

func clientPolicyTestCA(t *testing.T) (*x509.Certificate, *ecdsa.PrivateKey, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: "policy CA"},
		NotBefore:    time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
	require.NoError(t, err)
	parsed, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return parsed, key, string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}
