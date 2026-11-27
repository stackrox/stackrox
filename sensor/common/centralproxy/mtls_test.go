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
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	centralv1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func TestProfileTokenIsolationAndHeaderReplacement(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.CentralProxyPermissionSetTokens})
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
		identity: clientAuthMTLS + ":prometheus",
		permissionSets: []*centralv1.SelectedPermissionSet{{
			Selector: &centralv1.SelectedPermissionSet_Name{Name: accesscontrol.OpenShiftPrometheusMetricsReader},
		}},
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
	require.ErrorContains(t, err, "only available to console and authentication-only grants")
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
