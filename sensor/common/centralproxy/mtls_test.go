package centralproxy

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	centralv1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/env"
	pkghttputil "github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func testProxyCA(t *testing.T) (*x509.Certificate, *ecdsa.PrivateKey) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: "OpenShift test CA"},
		NotBefore:    time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	ca, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return ca, key
}

func signedTestCertificate(t *testing.T, ca *x509.Certificate, caKey *ecdsa.PrivateKey, cn string,
	usage x509.ExtKeyUsage, expiry ...time.Time,
) tls.Certificate {
	t.Helper()
	notAfter := time.Now().Add(time.Hour)
	if len(expiry) > 0 {
		notAfter = expiry[0]
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: cn}, DNSNames: []string{"localhost"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: notAfter,
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{usage},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, ca, &key.PublicKey, caKey)
	require.NoError(t, err)
	leaf, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return tls.Certificate{Certificate: [][]byte{der, ca.Raw}, PrivateKey: key, Leaf: leaf}
}

func clientCAConfigMap(ca *x509.Certificate) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: env.SecureMetricsClientCANamespace.Setting(), Name: env.SecureMetricsClientCAConfigMap.Setting(),
		},
		Data: map[string]string{
			env.SecureMetricsClientCAKey.Setting(): string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.Raw})),
		},
	}
}

func TestMetricsPolicyCertificateAuthentication(t *testing.T) {
	setupCentralCapsForTest(t)
	ca, key := testProxyCA(t)
	otherCA, otherKey := testProxyCA(t)
	cn := env.SecureMetricsClientCertCN.Setting()
	for name, tc := range map[string]struct {
		cert    *tls.Certificate
		path    string
		allowed bool
	}{
		"trusted client":   {cert: new(signedTestCertificate(t, ca, key, cn, x509.ExtKeyUsageClientAuth)), allowed: true},
		"wrong CN":         {cert: new(signedTestCertificate(t, ca, key, "other", x509.ExtKeyUsageClientAuth))},
		"untrusted signer": {cert: new(signedTestCertificate(t, otherCA, otherKey, cn, x509.ExtKeyUsageClientAuth))},
		"expired":          {cert: new(signedTestCertificate(t, ca, key, cn, x509.ExtKeyUsageClientAuth, time.Now().Add(-time.Minute)))},
		"wrong EKU":        {cert: new(signedTestCertificate(t, ca, key, cn, x509.ExtKeyUsageServerAuth))},
		"no certificate":   {},
		"certificate cannot fall back to console bearer": {
			cert: new(signedTestCertificate(t, ca, key, cn, x509.ExtKeyUsageClientAuth)), path: "/api/graphql",
		},
	} {
		t.Run(name, func(t *testing.T) {
			policy := newStaticClientPolicy(t, defaultClientPolicy(), clientCAConfigMap(ca))
			path := tc.path
			if path == "" {
				path = "/metrics"
			}
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.Header.Set("Authorization", "Bearer must-not-be-reviewed")
			if tc.cert != nil {
				req.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{tc.cert.Leaf}}
			}
			grant, err := policy.authorizeRequest(t.Context(), req, func() (*authenticationv1.UserInfo, error) {
				if tc.cert != nil {
					t.Fatal("certificate clients must never use bearer authentication")
				}
				return nil, errors.New("no reviewed bearer")
			})
			if err == nil {
				err = validateClientRequest(grant, req)
			}
			if !tc.allowed {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, accesscontrol.OpenShiftPrometheus, grant.PermissionSets[0].GetName())
			assert.Empty(t, grant.PermissionSets[0].GetId())
			assert.Equal(t, clientAuthMTLS, grant.AuthType)
			assert.Equal(t, cn, grant.Subject)
		})
	}
}

func TestMetricsOnlyPolicyDeniesConsole(t *testing.T) {
	t.Setenv("ROX_OCP_CONSOLE_INTEGRATION", "false")
	policy := newStaticClientPolicy(t, defaultClientPolicy())
	request := httptest.NewRequest(http.MethodGet, "/v1/metadata", nil)
	request.Header.Set("Authorization", "Bearer console-token")
	grant, err := policy.Authorize(t.Context(), request, &authenticationv1.UserInfo{Username: "alice"})
	require.NoError(t, err)
	err = validateClientRequest(grant, request)
	require.Error(t, err)
	assert.Equal(t, http.StatusForbidden, pkghttputil.StatusFromError(err))
}

func TestMetricsCAWatchRevokesExistingConnections(t *testing.T) {
	ca, key := testProxyCA(t)
	newCA, newKey := testProxyCA(t)
	clientCert := signedTestCertificate(t, ca, key, env.SecureMetricsClientCertCN.Setting(), x509.ExtKeyUsageClientAuth)
	rotatedClientCert := signedTestCertificate(t, newCA, newKey, env.SecureMetricsClientCertCN.Setting(), x509.ExtKeyUsageClientAuth)
	cm := clientCAConfigMap(ca)
	client := fake.NewClientset(cm)
	policy := NewClientPolicy(client)
	require.NoError(t, policy.EnsureDefaults(t.Context()))
	ref := clientPolicyCARef{Namespace: cm.Namespace, ConfigMap: cm.Name, Key: env.SecureMetricsClientCAKey.Setting()}
	caWatch := policy.policy.Load().caRoots[ref]
	require.NotNil(t, caWatch)
	config, updateServerCert := proxyTLSConfig(signedTestCertificate(t, ca, key, "localhost", x509.ExtKeyUsageServerAuth))
	policy.configureTLS(config)
	t.Cleanup(policy.Stop)
	assert.True(t, config.SessionTicketsDisabled)
	assert.Equal(t, tls.RequestClientCert, config.ClientAuth)
	require.Eventually(t, func() bool { return caWatch.roots.Load() != nil }, 5*time.Second, 10*time.Millisecond)
	var connections atomic.Int32
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := policy.Authorize(r.Context(), r, nil); err != nil {
			http.Error(w, err.Error(), pkghttputil.StatusFromError(err))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	server.Config.ConnContext = func(ctx context.Context, _ net.Conn) context.Context {
		connections.Add(1)
		return ctx
	}
	server.TLS = config
	server.StartTLS()
	t.Cleanup(server.Close)
	roots := x509.NewCertPool()
	roots.AddCert(ca)
	transport := &http.Transport{TLSClientConfig: &tls.Config{
		RootCAs: roots, Certificates: []tls.Certificate{clientCert}, ServerName: "localhost", MinVersion: tls.VersionTLS12,
	}}
	t.Cleanup(transport.CloseIdleConnections)
	httpClient := &http.Client{Transport: transport}
	scrape := func() int {
		response, err := httpClient.Get(server.URL + "/metrics")
		require.NoError(t, err)
		_, err = io.Copy(io.Discard, response.Body)
		require.NoError(t, err)
		require.NoError(t, response.Body.Close())
		return response.StatusCode
	}
	assert.Equal(t, http.StatusOK, scrape())
	configMaps := client.CoreV1().ConfigMaps(cm.Namespace)
	for name, data := range map[string]map[string]string{
		"missing CA key": {},
		"empty CA":       {env.SecureMetricsClientCAKey.Setting(): ""},
		"invalid PEM":    {env.SecureMetricsClientCAKey.Setting(): "invalid"},
	} {
		t.Run(name, func(t *testing.T) {
			invalid := cm.DeepCopy()
			invalid.Data = data
			_, err := configMaps.Update(t.Context(), invalid, metav1.UpdateOptions{})
			require.NoError(t, err)
			require.Eventually(t, func() bool { return caWatch.roots.Load() == nil }, 5*time.Second, 10*time.Millisecond)
			assert.Equal(t, http.StatusServiceUnavailable, scrape())
			_, err = configMaps.Update(t.Context(), cm, metav1.UpdateOptions{})
			require.NoError(t, err)
			require.Eventually(t, func() bool { return caWatch.roots.Load() != nil }, 5*time.Second, 10*time.Millisecond)
			assert.Equal(t, http.StatusOK, scrape())
		})
	}
	oldRoots := caWatch.roots.Load()
	_, err := configMaps.Update(t.Context(), clientCAConfigMap(newCA), metav1.UpdateOptions{})
	require.NoError(t, err)
	require.Eventually(t, func() bool { return caWatch.roots.Load() != oldRoots }, 5*time.Second, 10*time.Millisecond)
	assert.Equal(t, http.StatusUnauthorized, scrape())
	assert.EqualValues(t, 1, connections.Load(), "CA rotation must revoke a client on its existing TLS connection")

	rotatedTransport := transport.Clone()
	rotatedTransport.TLSClientConfig.Certificates = []tls.Certificate{rotatedClientCert}
	t.Cleanup(rotatedTransport.CloseIdleConnections)
	httpClient = &http.Client{Transport: rotatedTransport}
	assert.Equal(t, http.StatusOK, scrape())
	require.NoError(t, configMaps.Delete(t.Context(), cm.Name, metav1.DeleteOptions{}))
	require.Eventually(t, func() bool { return caWatch.roots.Load() == nil }, 5*time.Second, 10*time.Millisecond)
	assert.Equal(t, http.StatusServiceUnavailable, scrape())
	rotatedServerCert := signedTestCertificate(t, ca, key, "localhost", x509.ExtKeyUsageServerAuth)
	updateServerCert(&rotatedServerCert)
	live, err := config.GetConfigForClient(nil)
	require.NoError(t, err)
	assert.Equal(t, rotatedServerCert.Certificate, live.Certificates[0].Certificate)
}

func TestMetricsHandlerProfileAndDenials(t *testing.T) {
	ca, key := testProxyCA(t)
	cert := signedTestCertificate(t, ca, key, env.SecureMetricsClientCertCN.Setting(), x509.ExtKeyUsageClientAuth)
	for name, tc := range map[string]struct {
		path    string
		method  string
		capable bool
		want    int
	}{
		"metrics scrape":             {path: "/metrics", method: http.MethodGet, capable: true, want: http.StatusOK},
		"new capability required":    {path: "/metrics", method: http.MethodGet, want: http.StatusNotImplemented},
		"POST denied":                {path: "/metrics", method: http.MethodPost, capable: true, want: http.StatusForbidden},
		"HEAD denied":                {path: "/metrics", method: http.MethodHead, capable: true, want: http.StatusForbidden},
		"trailing slash denied":      {path: "/metrics/", method: http.MethodGet, capable: true, want: http.StatusForbidden},
		"encoded path denied":        {path: "/%6detrics", method: http.MethodGet, capable: true, want: http.StatusForbidden},
		"double encoded path denied": {path: "/%256detrics", method: http.MethodGet, capable: true, want: http.StatusForbidden},
		"dot segment denied":         {path: "/v1/../metrics", method: http.MethodGet, capable: true, want: http.StatusForbidden},
	} {
		t.Run(name, func(t *testing.T) {
			setupCentralCapsForTest(t)
			t.Setenv("ROX_OCP_CONSOLE_INTEGRATION", "false")
			caps := []centralsensor.CentralCapability{centralsensor.InternalTokenAPISupported, centralsensor.CentralProxyPathFiltering}
			if tc.capable {
				caps = append(caps, centralsensor.CentralProxyPermissionSetTokens, centralsensor.CentralProxyPerAccessTokens)
			}
			centralcaps.Set(caps)
			policy := newStaticClientPolicy(t, defaultClientPolicy(), clientCAConfigMap(ca))
			baseURL, err := url.Parse("https://central:443")
			require.NoError(t, err)
			var forwarded bool
			handler := &Handler{
				policy: policy,
				proxy: newReverseProxyForTest(baseURL, roundTripperFunc(func(r *http.Request) (*http.Response, error) {
					forwarded = true
					profile := proxyTokenProfileFromContext(r.Context())
					require.NotNil(t, profile)
					assert.Equal(t, accesscontrol.OpenShiftPrometheus, profile.permissionSets[0].GetName())
					assert.Equal(t, clientAuthMTLS+":"+cert.Leaf.Subject.CommonName, profile.identity)
					assert.Equal(t, FullClusterAccessScope, profile.clusterScope)
					assert.False(t, profile.legacy)
					return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(""))}, nil
				})),
			}
			handler.centralReachable.Store(true)
			request := httptest.NewRequest(tc.method, tc.path, nil)
			request.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert.Leaf}}
			request.Header.Set(stackroxNamespaceHeader, "untrusted-scope")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			assert.Equal(t, tc.want, response.Code)
			assert.Equal(t, tc.want == http.StatusOK, forwarded)
		})
	}
}

func TestProfileTokenIsolationAndHeaderReplacement(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.CentralProxyPermissionSetTokens, centralsensor.CentralProxyPerAccessTokens})
	t.Cleanup(func() { centralcaps.Set(nil) })
	var calls atomic.Int32
	client := &fakeTokenServiceClient{response: &centralv1.GenerateTokenForPermissionsAndScopeResponse{Token: "central-token"}, callCount: &calls}
	provider := newTestTokenProvider(client, "sensor-cluster")
	console := consoleTestProfile("prod")
	for name, change := range map[string]func(*proxyTokenProfile){
		"identity": func(p *proxyTokenProfile) { p.identity = "another-client" },
		"scope":    func(p *proxyTokenProfile) { p.clusterScope = "another-namespace" },
		"selector": func(p *proxyTokenProfile) {
			p.permissionSets = []*centralv1.SelectedPermissionSet{{Selector: &centralv1.SelectedPermissionSet_Name{Name: "Reader"}}}
		},
		"authentication-only": func(p *proxyTokenProfile) { p.permissionSets = nil },
		"legacy mode":         func(p *proxyTokenProfile) { p.legacy = true },
	} {
		t.Run(name, func(t *testing.T) {
			other := *console
			change(&other)
			var issued atomic.Int32
			client := &fakeTokenServiceClient{
				response:  &centralv1.GenerateTokenForPermissionsAndScopeResponse{Token: "first"},
				callCount: &issued,
			}
			provider := newTestTokenProvider(client, "sensor-cluster")
			first, err := provider.getTokenForProfile(t.Context(), console)
			require.NoError(t, err)
			client.response = &centralv1.GenerateTokenForPermissionsAndScopeResponse{Token: "second"}
			second, err := provider.getTokenForProfile(t.Context(), &other)
			require.NoError(t, err)
			assert.NotEqual(t, first, second)
			firstAgain, err := provider.getTokenForProfile(t.Context(), console)
			require.NoError(t, err)
			assert.Equal(t, first, firstAgain)
			assert.EqualValues(t, 2, issued.Load())
		})
	}
	metrics := &proxyTokenProfile{
		identity: clientAuthMTLS + ":prometheus", permissionSets: clientPolicyAccessToProto(defaultClientPolicy().Clients[0].Access),
		clusterScope: FullClusterAccessScope,
	}
	transport := &scopedTokenTransport{tokenProvider: provider, base: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "Bearer central-token", r.Header.Get("Authorization"))
		assert.Empty(t, r.Header.Get(stackroxNamespaceHeader))
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
	})}
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer kubernetes-secret")
	req.Header.Set(stackroxNamespaceHeader, "untrusted")
	response, err := transport.RoundTrip(req)
	require.ErrorContains(t, err, "profile is missing")
	assert.Nil(t, response)
	req = req.WithContext(withProxyTokenProfile(req.Context(), metrics))
	response, err = transport.RoundTrip(req)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	assert.Equal(t, "Bearer kubernetes-secret", req.Header.Get("Authorization"))
	assert.Equal(t, metrics.identity, client.lastRequest.GetRequester())
	require.Len(t, client.lastRequest.GetPermissionSets(), 1)
	assert.Equal(t, metrics.permissionSets[0].GetName(), client.lastRequest.GetPermissionSets()[0].GetName())
	assert.Empty(t, client.lastRequest.GetPermissions())
	require.Len(t, client.lastRequest.GetClusterScopes(), 1)
	assert.Equal(t, "sensor-cluster", client.lastRequest.GetClusterScopes()[0].GetClusterId())
	assert.True(t, client.lastRequest.GetClusterScopes()[0].GetFullClusterAccess())
	assert.Equal(t, tokenTTL, client.lastRequest.GetLifetime().AsDuration())
	_, err = provider.getTokenForProfile(t.Context(), metrics)
	require.NoError(t, err)
	assert.EqualValues(t, 1, calls.Load())
	provider.clusterIDGetter = &fakeClusterIDGetter{clusterID: "another-cluster"}
	_, err = provider.getTokenForProfile(t.Context(), metrics)
	require.NoError(t, err)
	assert.EqualValues(t, 2, calls.Load())
	assert.Equal(t, "another-cluster", client.lastRequest.GetClusterScopes()[0].GetClusterId())
	legacyMetrics := *metrics
	legacyMetrics.legacy = true
	_, err = provider.getTokenForProfile(t.Context(), &legacyMetrics)
	require.ErrorContains(t, err, "only available to safely scoped console clients")
}

type tokenServiceFunc func(context.Context, *centralv1.GenerateTokenForPermissionsAndScopeRequest) (*centralv1.GenerateTokenForPermissionsAndScopeResponse, error)

func (f tokenServiceFunc) GenerateTokenForPermissionsAndScope(ctx context.Context,
	req *centralv1.GenerateTokenForPermissionsAndScopeRequest, _ ...grpc.CallOption,
) (*centralv1.GenerateTokenForPermissionsAndScopeResponse, error) {
	return f(ctx, req)
}

func TestProfileTokenRPCOutlivesCallerWithBoundedDeadline(t *testing.T) {
	started := make(chan context.Context, 1)
	release := make(chan struct{})
	var issued atomic.Int32
	provider := newTestTokenProvider(tokenServiceFunc(func(ctx context.Context, _ *centralv1.GenerateTokenForPermissionsAndScopeRequest) (*centralv1.GenerateTokenForPermissionsAndScopeResponse, error) {
		issued.Add(1)
		started <- ctx
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-release:
			return &centralv1.GenerateTokenForPermissionsAndScopeResponse{Token: "shared-token"}, nil
		}
	}), "sensor-cluster")
	profile := consoleTestProfile("prod")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, err := provider.getTokenForProfile(ctx, profile)
		result <- err
	}()
	rpcCtx := <-started
	deadline, hasDeadline := rpcCtx.Deadline()
	assert.True(t, hasDeadline)
	assert.WithinDuration(t, time.Now().Add(tokenRequestTimeout), deadline, time.Second)
	cancel()
	require.ErrorIs(t, <-result, context.Canceled)
	assert.NoError(t, rpcCtx.Err(), "one caller must not cancel the shared token RPC")
	close(release)
	token, err := provider.getTokenForProfile(t.Context(), profile)
	require.NoError(t, err)
	assert.Equal(t, "shared-token", token)
	assert.EqualValues(t, 1, issued.Load())
}
