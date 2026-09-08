package aisummary

import (
	"encoding/json"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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

	contextJSON, err := BuildSanitizedRiskContext(deployment, risk)
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
