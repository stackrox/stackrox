package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	olsClient "github.com/stackrox/rox/central/lightspeed/client"
	riskMocks "github.com/stackrox/rox/central/risk/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type mockOLSClient struct {
	response *olsClient.QueryResponse
	err      error
	captured *olsClient.QueryRequest
}

func (m *mockOLSClient) Query(_ context.Context, req *olsClient.QueryRequest) (*olsClient.QueryResponse, error) {
	m.captured = req
	return m.response, m.err
}

func TestGetDeploymentRiskAISummary_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockDS := deploymentMocks.NewMockDataStore(ctrl)
	mockRisks := riskMocks.NewMockDataStore(ctrl)

	deployment := &storage.Deployment{
		Id:                            "dep-1",
		Name:                          "nginx-frontend",
		Namespace:                     "production",
		ClusterName:                   "cluster-east",
		Type:                          "Deployment",
		Created:                       timestamppb.Now(),
		ServiceAccount:                "default",
		ServiceAccountPermissionLevel: storage.PermissionLevel_CLUSTER_ADMIN,
		AutomountServiceAccountToken:  true,
		HostNetwork:                   false,
		Containers: []*storage.Container{
			{
				Name: "nginx",
				Image: &storage.ContainerImage{
					Name: &storage.ImageName{FullName: "docker.io/nginx:1.25"},
				},
				SecurityContext: &storage.SecurityContext{
					Privileged:               true,
					AllowPrivilegeEscalation: true,
				},
				Resources: &storage.Resources{
					CpuCoresRequest: 0.5,
					CpuCoresLimit:   1.0,
					MemoryMbRequest: 256,
					MemoryMbLimit:   512,
				},
				Config: &storage.ContainerConfig{
					Uid: 0,
					Env: []*storage.ContainerConfig_EnvironmentConfig{
						{Key: "SECRET_KEY", Value: "super-secret-123"},
						{Key: "DB_URL", Value: "postgres://db.internal:5432/mydb"}, // #nosec G101
					},
				},
				LivenessProbe:  &storage.LivenessProbe{Defined: true},
				ReadinessProbe: &storage.ReadinessProbe{Defined: false},
				Secrets: []*storage.EmbeddedSecret{
					{Name: "db-creds", Path: "/var/run/secrets/db"},
				},
			},
		},
	}

	risk := &storage.Risk{
		Id:    "risk-1",
		Score: 8.5,
		Subject: &storage.RiskSubject{
			Id:        "dep-1",
			Namespace: "production",
			ClusterId: "cluster-1",
			Type:      storage.RiskSubjectType_DEPLOYMENT,
		},
		Results: []*storage.Risk_Result{
			{
				Name:  "Service Configuration",
				Score: 3.5,
				Factors: []*storage.Risk_Result_Factor{
					{Message: "Service account is cluster-admin"},
					{Message: "Automount SA token enabled"},
				},
			},
			{
				Name:  "Image Vulnerabilities",
				Score: 2.0,
				Factors: []*storage.Risk_Result_Factor{
					{Message: "Image has 42 CVEs", Url: "https://example.com/cves"},
				},
			},
		},
	}

	mockDS.EXPECT().GetDeployment(gomock.Any(), "dep-1").Return(deployment, true, nil)
	mockRisks.EXPECT().GetRiskForDeployment(gomock.Any(), deployment).Return(risk, true, nil)

	olsMock := &mockOLSClient{
		response: &olsClient.QueryResponse{
			Response: "SUMMARY\nThis deployment runs as cluster-admin with a privileged container.",
		},
	}

	svc := &serviceImpl{
		datastore:        mockDS,
		risks:            mockRisks,
		lightspeedClient: olsMock,
	}

	resp, err := svc.GetDeploymentRiskAISummary(context.Background(), &v1.ResourceByID{Id: "dep-1"})
	require.NoError(t, err)
	assert.Equal(t, "SUMMARY\nThis deployment runs as cluster-admin with a privileged container.", resp.GetSummary())

	// Verify prompt and context were combined into the query.
	assert.Contains(t, olsMock.captured.Query, aiSummaryPrompt)
	assert.Contains(t, olsMock.captured.Query, "DEPLOYMENT AND RISK DATA:")
	assert.Contains(t, olsMock.captured.Query, "nginx-frontend")
}

func TestGetDeploymentRiskAISummary_SensitiveFieldsStripped(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockDS := deploymentMocks.NewMockDataStore(ctrl)
	mockRisks := riskMocks.NewMockDataStore(ctrl)

	deployment := &storage.Deployment{
		Id:          "dep-2",
		Name:        "mid-server",
		Namespace:   "operations",
		ClusterName: "cluster-west",
		Type:        "Deployment",
		Containers: []*storage.Container{
			{
				Name: "app",
				Image: &storage.ContainerImage{
					Id:   "sha256:abc123",
					IdV2: "img-v2-id",
					Name: &storage.ImageName{FullName: "registry.example.com/app:v2"},
				},
				Config: &storage.ContainerConfig{
					Uid: 1000,
					Env: []*storage.ContainerConfig_EnvironmentConfig{
						{Key: "SERVICENOW_URL", Value: "https://company.service-now.com"},
						{Key: "SN_USER", Value: "admin"},
						{Key: "SN_SECRET_REF", Value: "vault:secret/data/sn-key"},
					},
					Command:   []string{"/bin/app"},
					Args:      []string{"--port=8080"},
					Directory: "/app",
				},
				Secrets: []*storage.EmbeddedSecret{
					{Name: "sn-credentials", Path: "/var/run/secrets/servicenow"},
				},
				SecurityContext: &storage.SecurityContext{
					Privileged: false,
				},
			},
		},
	}

	risk := &storage.Risk{
		Id:    "risk-2",
		Score: 4.0,
		Subject: &storage.RiskSubject{
			Id:        "dep-2",
			Namespace: "operations",
		},
		Results: []*storage.Risk_Result{
			{
				Name:  "Image Freshness",
				Score: 2.0,
				Factors: []*storage.Risk_Result_Factor{
					{Message: "Image is 180 days old", Url: ""},
				},
			},
		},
	}

	mockDS.EXPECT().GetDeployment(gomock.Any(), "dep-2").Return(deployment, true, nil)
	mockRisks.EXPECT().GetRiskForDeployment(gomock.Any(), deployment).Return(risk, true, nil)

	olsMock := &mockOLSClient{
		response: &olsClient.QueryResponse{Response: "Low risk deployment."},
	}

	svc := &serviceImpl{
		datastore:        mockDS,
		risks:            mockRisks,
		lightspeedClient: olsMock,
	}

	_, err := svc.GetDeploymentRiskAISummary(context.Background(), &v1.ResourceByID{Id: "dep-2"})
	require.NoError(t, err)

	capturedQuery := olsMock.captured.Query

	// Sensitive env vars must NOT appear in query sent to LLM.
	assert.NotContains(t, capturedQuery, "SERVICENOW_URL")
	assert.NotContains(t, capturedQuery, "https://company.service-now.com")
	assert.NotContains(t, capturedQuery, "SN_USER")
	assert.NotContains(t, capturedQuery, "SN_SECRET_REF")
	assert.NotContains(t, capturedQuery, "vault:secret/data/sn-key")

	// Secret mount paths must NOT appear.
	assert.NotContains(t, capturedQuery, "/var/run/secrets/servicenow")
	assert.NotContains(t, capturedQuery, "sn-credentials")

	// Internal IDs must NOT appear.
	assert.NotContains(t, capturedQuery, "sha256:abc123")
	assert.NotContains(t, capturedQuery, "img-v2-id")
	assert.NotContains(t, capturedQuery, "dep-2")  // deployment ID
	assert.NotContains(t, capturedQuery, "risk-2") // risk ID

	// Command, args, and directory must NOT appear.
	assert.NotContains(t, capturedQuery, "/bin/app")
	assert.NotContains(t, capturedQuery, "--port=8080")

	// Relevant fields MUST appear.
	assert.Contains(t, capturedQuery, "mid-server")
	assert.Contains(t, capturedQuery, "operations")
	assert.Contains(t, capturedQuery, "cluster-west")
	assert.Contains(t, capturedQuery, "registry.example.com/app:v2")
	assert.Contains(t, capturedQuery, "Image is 180 days old")
}

func TestGetDeploymentRiskAISummary_DeploymentNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockDS := deploymentMocks.NewMockDataStore(ctrl)

	mockDS.EXPECT().GetDeployment(gomock.Any(), "nonexistent").Return(nil, false, nil)

	svc := &serviceImpl{
		datastore: mockDS,
	}

	resp, err := svc.GetDeploymentRiskAISummary(context.Background(), &v1.ResourceByID{Id: "nonexistent"})
	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestGetDeploymentRiskAISummary_OLSError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockDS := deploymentMocks.NewMockDataStore(ctrl)
	mockRisks := riskMocks.NewMockDataStore(ctrl)

	deployment := &storage.Deployment{
		Id:          "dep-3",
		Name:        "test-app",
		Namespace:   "default",
		ClusterName: "test-cluster",
		Type:        "Deployment",
	}
	risk := &storage.Risk{Score: 2.0}

	mockDS.EXPECT().GetDeployment(gomock.Any(), "dep-3").Return(deployment, true, nil)
	mockRisks.EXPECT().GetRiskForDeployment(gomock.Any(), deployment).Return(risk, true, nil)

	olsMock := &mockOLSClient{
		err: errors.New("Lightspeed API returned HTTP 503"),
	}

	svc := &serviceImpl{
		datastore:        mockDS,
		risks:            mockRisks,
		lightspeedClient: olsMock,
	}

	resp, err := svc.GetDeploymentRiskAISummary(context.Background(), &v1.ResourceByID{Id: "dep-3"})
	assert.Nil(t, resp)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok, "expected a gRPC status error")
	assert.Equal(t, codes.Unavailable, st.Code())
	assert.Equal(t, "AI service unavailable", st.Message())
}

func TestGetDeploymentRiskAISummary_NilRisk(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockDS := deploymentMocks.NewMockDataStore(ctrl)
	mockRisks := riskMocks.NewMockDataStore(ctrl)

	deployment := &storage.Deployment{
		Id:          "dep-4",
		Name:        "new-app",
		Namespace:   "staging",
		ClusterName: "staging-cluster",
		Type:        "Deployment",
	}

	mockDS.EXPECT().GetDeployment(gomock.Any(), "dep-4").Return(deployment, true, nil)
	mockRisks.EXPECT().GetRiskForDeployment(gomock.Any(), deployment).Return(nil, false, nil)

	olsMock := &mockOLSClient{
		response: &olsClient.QueryResponse{Response: "No significant risk factors identified."},
	}

	svc := &serviceImpl{
		datastore:        mockDS,
		risks:            mockRisks,
		lightspeedClient: olsMock,
	}

	resp, err := svc.GetDeploymentRiskAISummary(context.Background(), &v1.ResourceByID{Id: "dep-4"})
	require.NoError(t, err)
	assert.Equal(t, "No significant risk factors identified.", resp.GetSummary())
}

func TestBuildSanitizedRiskContext_FieldSelection(t *testing.T) {
	deployment := &storage.Deployment{
		Id:                            "id-should-be-stripped",
		Hash:                          12345,
		Name:                          "my-deployment",
		Namespace:                     "my-namespace",
		NamespaceId:                   "ns-id-should-be-stripped",
		ClusterId:                     "cluster-id-should-be-stripped",
		ClusterName:                   "my-cluster",
		Type:                          "Deployment",
		Created:                       timestamppb.Now(),
		ServiceAccount:                "deployer-sa",
		ServiceAccountPermissionLevel: storage.PermissionLevel_ELEVATED_IN_NAMESPACE,
		AutomountServiceAccountToken:  true,
		HostNetwork:                   true,
		HostPid:                       false,
		HostIpc:                       false,
		StateTimestamp:                999999,
		ImagePullSecrets:              []string{"registry-secret"},
		Tolerations: []*storage.Toleration{
			{Key: "node-role.kubernetes.io/master"},
		},
		Ports: []*storage.PortConfig{
			{Name: "http", ContainerPort: 8080},
		},
		Labels:      map[string]string{"app": "web"},
		Annotations: map[string]string{"note": "test"},
		Containers: []*storage.Container{
			{
				Id:   "container-id-should-be-stripped",
				Name: "web",
				Image: &storage.ContainerImage{
					Id:             "sha256:deadbeef",
					IdV2:           "idv2-stripped",
					Name:           &storage.ImageName{FullName: "quay.io/app:latest"},
					NotPullable:    true,
					IsClusterLocal: true,
				},
				SecurityContext: &storage.SecurityContext{
					Privileged:               true,
					ReadOnlyRootFilesystem:   true,
					AllowPrivilegeEscalation: false,
					DropCapabilities:         []string{"ALL"},
					AddCapabilities:          []string{"NET_BIND_SERVICE"},
				},
				Resources: &storage.Resources{
					CpuCoresRequest: 0.25,
					CpuCoresLimit:   1.0,
					MemoryMbRequest: 128,
					MemoryMbLimit:   512,
				},
				Config: &storage.ContainerConfig{
					Uid: 0,
					Env: []*storage.ContainerConfig_EnvironmentConfig{
						{Key: "PASSWORD", Value: "secret123"},
					},
					Command:   []string{"/entrypoint.sh"},
					Directory: "/app",
				},
				LivenessProbe:  &storage.LivenessProbe{Defined: true},
				ReadinessProbe: &storage.ReadinessProbe{Defined: true},
				Secrets: []*storage.EmbeddedSecret{
					{Name: "tls-cert", Path: "/etc/tls/certs"},
				},
			},
		},
	}

	risk := &storage.Risk{
		Id:    "risk-id-stripped",
		Score: 7.2,
		Subject: &storage.RiskSubject{
			Id:        "dep-subject",
			Namespace: "my-namespace",
			ClusterId: "cluster-1",
		},
		Results: []*storage.Risk_Result{
			{
				Name:  "Policy Violations",
				Score: 4.0,
				Factors: []*storage.Risk_Result_Factor{
					{Message: "Privileged container detected", Url: "https://docs.example.com/priv"},
					{Message: "Running as root"},
				},
			},
		},
	}

	contextJSON, err := buildSanitizedRiskContext(deployment, risk)
	require.NoError(t, err)

	// Parse to verify structure.
	var parsed map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(contextJSON), &parsed))
	assert.Contains(t, string(parsed["deployment"]), "my-deployment")
	assert.Contains(t, string(parsed["risk"]), "Policy Violations")

	// Fields that MUST be present.
	assert.Contains(t, contextJSON, "my-deployment")
	assert.Contains(t, contextJSON, "my-namespace")
	assert.Contains(t, contextJSON, "my-cluster")
	assert.Contains(t, contextJSON, "Deployment")
	assert.Contains(t, contextJSON, "deployer-sa")
	assert.Contains(t, contextJSON, "ELEVATED_IN_NAMESPACE")
	assert.Contains(t, contextJSON, "quay.io/app:latest")
	assert.Contains(t, contextJSON, "Privileged container detected")
	assert.Contains(t, contextJSON, "Running as root")

	// Fields that MUST be stripped — IDs and metadata.
	assert.NotContains(t, contextJSON, "id-should-be-stripped")
	assert.NotContains(t, contextJSON, "ns-id-should-be-stripped")
	assert.NotContains(t, contextJSON, "cluster-id-should-be-stripped")
	assert.NotContains(t, contextJSON, "container-id-should-be-stripped")
	assert.NotContains(t, contextJSON, "sha256:deadbeef")
	assert.NotContains(t, contextJSON, "idv2-stripped")
	assert.NotContains(t, contextJSON, "risk-id-stripped")
	assert.NotContains(t, contextJSON, "dep-subject")
	assert.NotContains(t, contextJSON, "999999") // stateTimestamp

	// Fields stripped for security.
	assert.NotContains(t, contextJSON, "PASSWORD")
	assert.NotContains(t, contextJSON, "secret123")
	assert.NotContains(t, contextJSON, "/etc/tls/certs")
	assert.NotContains(t, contextJSON, "tls-cert")
	assert.NotContains(t, contextJSON, "/entrypoint.sh")

	// Wasteful fields stripped.
	assert.NotContains(t, contextJSON, "registry-secret")                // imagePullSecrets
	assert.NotContains(t, contextJSON, "node-role.kubernetes.io/master") // tolerations
	assert.NotContains(t, contextJSON, "https://docs.example.com/priv")  // factor URLs

	// notPullable and isClusterLocal should not be included.
	assert.NotContains(t, contextJSON, "notPullable")
	assert.NotContains(t, contextJSON, "isClusterLocal")
}
