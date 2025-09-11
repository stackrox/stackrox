package extensions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/common/rendercache"
	"github.com/stackrox/rox/operator/internal/utils/testutils"
	"github.com/stackrox/rox/pkg/crs"
	"github.com/stackrox/rox/pkg/securedcluster"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testNamespace = "test-namespace"
)

func TestSensorCAHashExtension_CertificatePriority(t *testing.T) {
	tests := []struct {
		name         string
		secrets      []ctrlClient.Object
		expectedHash string
		expectError  bool
	}{
		{
			name: "Priority 1: runtime certificates (tls-cert-sensor) preferred over others",
			secrets: append([]ctrlClient.Object{
				createTLSSecret("tls-cert-sensor", "ca1-content"),
				createCRSSecret(t, "ca2-content"),
				createTLSSecret("sensor-tls", "ca3-content"),
			}, createAllTLSSecrets("ca1-content")...),
			expectedHash: hashString("ca1-content"),
		},
		{
			name: "Priority 2: cluster-registration-secret when tls-cert-sensor missing",
			secrets: []ctrlClient.Object{
				createCRSSecret(t, "ca2-content"),
				createTLSSecret("sensor-tls", "ca3-content"),
			},
			expectedHash: hashString("ca2-content"),
		},
		{
			name: "Priority 3: init bundle (sensor-tls) fallback when others missing",
			secrets: []ctrlClient.Object{
				createTLSSecret("sensor-tls", "ca3-content"),
			},
			expectedHash: hashString("ca3-content"),
		},
		{
			name:        "No secrets: should error",
			secrets:     []ctrlClient.Object{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := testutils.NewFakeClientBuilder(t, tt.secrets...).Build()
			renderCache := rendercache.NewRenderCache()
			sc := createTestSecuredCluster()
			scUnstructured := toUnstructured(t, sc)

			extension := SensorCAHashExtension(client, client, logr.Discard(), renderCache)
			err := extension(context.Background(), scUnstructured, nil, logr.Discard())

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Verify the hash of the expected CA was stored in RenderCache
			cachedHash, found := renderCache.GetCAHash(sc)
			require.True(t, found, "CA hash should be cached")
			assert.Equal(t, tt.expectedHash, cachedHash)
		})
	}
}

func TestSensorCAHashExtension_ConsistencyCheck(t *testing.T) {
	tests := []struct {
		name        string
		secrets     []ctrlClient.Object
		expectError bool
		errorMsg    string
	}{
		{
			name: "All TLS secrets consistent with tls-cert-sensor",
			secrets: append(
				[]ctrlClient.Object{createTLSSecret("tls-cert-sensor", "ca1-content")},
				createAllTLSSecrets("ca1-content")...,
			),
			expectError: false,
		},
		{
			name: "Inconsistent TLS secrets should fail",
			secrets: []ctrlClient.Object{
				createTLSSecret("tls-cert-sensor", "ca1-content"),
				createTLSSecret("tls-cert-collector", "ca1-content"),
				createTLSSecret("tls-cert-admission-control", "ca2-content"), // Different CA
				createTLSSecret("tls-cert-scanner", "ca1-content"),
				createTLSSecret("tls-cert-scanner-db", "ca1-content"),
				createTLSSecret("tls-cert-scanner-v4-indexer", "ca1-content"),
				createTLSSecret("tls-cert-scanner-v4-db", "ca1-content"),
			},
			expectError: true,
			errorMsg:    "TLS secrets are not consistent",
		},
		{
			name: "Missing TLS secrets should fail consistency check",
			secrets: []ctrlClient.Object{
				createTLSSecret("tls-cert-sensor", "ca1-content"),
				createTLSSecret("tls-cert-collector", "ca1-content"),
			},
			expectError: true,
			errorMsg:    "TLS secrets are not consistent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := testutils.NewFakeClientBuilder(t, tt.secrets...).Build()
			renderCache := rendercache.NewRenderCache()
			sc := createTestSecuredCluster()
			scUnstructured := toUnstructured(t, sc)

			extension := SensorCAHashExtension(client, client, logr.Discard(), renderCache)
			err := extension(context.Background(), scUnstructured, nil, logr.Discard())

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSensorCAHashExtension_DeletionCleanup(t *testing.T) {
	secrets := append([]ctrlClient.Object{
		createTLSSecret("tls-cert-sensor", "ca1-content"),
	}, createAllTLSSecrets("ca1-content")...)

	client := testutils.NewFakeClientBuilder(t, secrets...).Build()
	renderCache := rendercache.NewRenderCache()

	sc := createTestSecuredCluster()
	scUnstructured := toUnstructured(t, sc)

	extension := SensorCAHashExtension(client, client, logr.Discard(), renderCache)

	err := extension(context.Background(), scUnstructured, nil, logr.Discard())
	require.NoError(t, err)

	_, found := renderCache.GetCAHash(sc)
	assert.True(t, found, "CA hash should be cached after normal run")

	// Simulate CR deletion by setting DeletionTimestamp
	now := metav1.Now()
	scUnstructured.SetDeletionTimestamp(&now)

	err = extension(context.Background(), scUnstructured, nil, logr.Discard())
	require.NoError(t, err)

	// Verify cache entry was removed after CR deletion
	_, found = renderCache.GetCAHash(sc)
	assert.False(t, found, "CA hash should be removed from cache after deletion")
}

func createTestSecuredCluster() *platform.SecuredCluster {
	return &platform.SecuredCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "platform.stackrox.io/v1alpha1",
			Kind:       "SecuredCluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secured-cluster",
			Namespace: testNamespace,
		},
	}
}

func createTLSSecret(name, caContent string) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			"ca.pem": []byte(caContent),
		},
	}
}

func createAllTLSSecrets(caContent string) []ctrlClient.Object {
	var secrets []ctrlClient.Object
	for _, name := range securedcluster.AllTLSSecretNames {
		secrets = append(secrets, createTLSSecret(name, caContent))
	}
	return secrets
}

func createCRSSecret(t *testing.T, caContent string) *v1.Secret {
	testCRS := &crs.CRS{
		Version: 1,
		CAs:     []string{caContent},
		Cert:    "test-cert",
		Key:     "test-key",
	}

	serializedCRS, err := crs.SerializeSecret(testCRS)
	require.NoError(t, err)

	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-registration-secret",
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			"crs": []byte(serializedCRS),
		},
	}
}

func hashString(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func toUnstructured(t *testing.T, obj ctrlClient.Object) *unstructured.Unstructured {
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	require.NoError(t, err)
	return &unstructured.Unstructured{Object: unstructuredObj}
}
