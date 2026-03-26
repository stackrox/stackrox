package discover

import (
	"context"
	"errors"
	"testing"

	"github.com/stackrox/co-acs-importer/internal/models"
)

// mockK8sClient implements the k8sResourceReader interface for testing.
type mockK8sClient struct {
	admissionControlCM    map[string]string
	admissionControlErr   error
	clusterVersionID      string
	clusterVersionErr     error
	helmSecretClusterName string
	helmSecretErr         error
}

func (m *mockK8sClient) getAdmissionControlClusterID(ctx context.Context) (string, error) {
	if m.admissionControlErr != nil {
		return "", m.admissionControlErr
	}
	return m.admissionControlCM["cluster-id"], nil
}

func (m *mockK8sClient) getOpenShiftClusterID(ctx context.Context) (string, error) {
	if m.clusterVersionErr != nil {
		return "", m.clusterVersionErr
	}
	return m.clusterVersionID, nil
}

func (m *mockK8sClient) getHelmSecretClusterName(ctx context.Context) (string, error) {
	if m.helmSecretErr != nil {
		return "", m.helmSecretErr
	}
	return m.helmSecretClusterName, nil
}

// mockACSClient implements the models.ACSClient interface for testing.
type mockACSClient struct {
	clusters []models.ACSClusterInfo
	err      error
}

func (m *mockACSClient) Preflight(ctx context.Context) error {
	return nil
}

func (m *mockACSClient) ListScanConfigurations(ctx context.Context) ([]models.ACSConfigSummary, error) {
	return nil, nil
}

func (m *mockACSClient) CreateScanConfiguration(ctx context.Context, payload models.ACSCreatePayload) (string, error) {
	return "", nil
}

func (m *mockACSClient) UpdateScanConfiguration(ctx context.Context, id string, payload models.ACSCreatePayload) error {
	return nil
}

func (m *mockACSClient) ListClusters(ctx context.Context) ([]models.ACSClusterInfo, error) {
	return m.clusters, m.err
}

// TestIMP_MAP_016_AdmissionControlConfigMap verifies discovery via admission-control ConfigMap.
func TestIMP_MAP_016_AdmissionControlConfigMap(t *testing.T) {
	ctx := context.Background()
	k8s := &mockK8sClient{
		admissionControlCM: map[string]string{"cluster-id": "acs-uuid-12345"},
	}
	acs := &mockACSClient{}

	clusterID, err := DiscoverClusterID(ctx, k8s, acs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clusterID != "acs-uuid-12345" {
		t.Errorf("expected cluster ID from admission-control CM, got %q", clusterID)
	}
}

// TestIMP_MAP_017_OpenShiftClusterVersion verifies discovery via OpenShift ClusterVersion.
func TestIMP_MAP_017_OpenShiftClusterVersion(t *testing.T) {
	ctx := context.Background()
	k8s := &mockK8sClient{
		admissionControlErr: errors.New("not found"),
		clusterVersionID:    "ocp-cluster-abc",
	}
	acs := &mockACSClient{
		clusters: []models.ACSClusterInfo{
			{ID: "acs-uuid-1", Name: "cluster-1", ProviderClusterID: "ocp-cluster-abc"},
			{ID: "acs-uuid-2", Name: "cluster-2", ProviderClusterID: "ocp-cluster-xyz"},
		},
	}

	clusterID, err := DiscoverClusterID(ctx, k8s, acs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clusterID != "acs-uuid-1" {
		t.Errorf("expected cluster ID from OpenShift ClusterVersion match, got %q", clusterID)
	}
}

// TestIMP_MAP_018_HelmSecretClusterName verifies discovery via helm-effective-cluster-name secret.
func TestIMP_MAP_018_HelmSecretClusterName(t *testing.T) {
	ctx := context.Background()
	k8s := &mockK8sClient{
		admissionControlErr:   errors.New("not found"),
		clusterVersionErr:     errors.New("not found"),
		helmSecretClusterName: "production",
	}
	acs := &mockACSClient{
		clusters: []models.ACSClusterInfo{
			{ID: "acs-uuid-1", Name: "production"},
			{ID: "acs-uuid-2", Name: "staging"},
		},
	}

	clusterID, err := DiscoverClusterID(ctx, k8s, acs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clusterID != "acs-uuid-1" {
		t.Errorf("expected cluster ID from helm secret match, got %q", clusterID)
	}
}

// TestDiscoveryFallbackChain verifies the discovery chain tries methods in order.
func TestDiscoveryFallbackChain(t *testing.T) {
	ctx := context.Background()
	k8s := &mockK8sClient{
		admissionControlErr:   errors.New("not found"),
		clusterVersionErr:     errors.New("not found"),
		helmSecretClusterName: "fallback-cluster",
	}
	acs := &mockACSClient{
		clusters: []models.ACSClusterInfo{
			{ID: "acs-uuid-fallback", Name: "fallback-cluster"},
		},
	}

	clusterID, err := DiscoverClusterID(ctx, k8s, acs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clusterID != "acs-uuid-fallback" {
		t.Errorf("expected fallback cluster ID, got %q", clusterID)
	}
}

// TestDiscoveryAllMethodsFail verifies error when all discovery methods fail.
func TestDiscoveryAllMethodsFail(t *testing.T) {
	ctx := context.Background()
	k8s := &mockK8sClient{
		admissionControlErr: errors.New("not found"),
		clusterVersionErr:   errors.New("not found"),
		helmSecretErr:       errors.New("not found"),
	}
	acs := &mockACSClient{}

	_, err := DiscoverClusterID(ctx, k8s, acs)
	if err == nil {
		t.Fatal("expected error when all discovery methods fail, got nil")
	}
}
