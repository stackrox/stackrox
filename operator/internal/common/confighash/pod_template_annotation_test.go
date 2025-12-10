package confighash

import (
	"bytes"
	"strings"
	"testing"

	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/common/rendercache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/resource"
)

const (
	testConfigHash = "test-config-hash-12345"
	testNamespace  = "test-namespace"
)

func TestRunWithEmptyCAHash(t *testing.T) {
	central := &platform.Central{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-central",
			Namespace: testNamespace,
			UID:       types.UID("test-uid"),
		},
	}

	pr := podTemplateAnnotationPostRenderer{
		renderCache: rendercache.NewRenderCache(),
		obj:         central,
	}

	originalManifest := "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: test"
	inputBuffer := bytes.NewBufferString(originalManifest)

	result, err := pr.Run(inputBuffer)
	require.NoError(t, err)

	assert.Equal(t, originalManifest, result.String(), "Buffer content should be unchanged when CA hash is empty")
}

func TestApplyPodTemplateAnnotationAndSerialize(t *testing.T) {
	deployment := createTestDeployment()
	daemonSet := createTestDaemonSet()
	replicaSet := createTestReplicaSet()

	resourceList := kube.ResourceList{
		&resource.Info{Object: deployment, Name: "central", Namespace: testNamespace},
		&resource.Info{Object: daemonSet, Name: "collector", Namespace: testNamespace},
		&resource.Info{Object: replicaSet, Name: "scanner", Namespace: testNamespace},
	}

	output, err := applyPodTemplateAnnotationAndSerialize(resourceList, testConfigHash)
	require.NoError(t, err)
	outputStr := output.String()

	// Check that the Deployment and the DaemonSet have the config-hash annotation
	deploymentSection := extractResourceSection(outputStr, "kind: Deployment")
	assert.Contains(t, deploymentSection, AnnotationKey+": "+testConfigHash, "Deployment should have config-hash annotation")
	daemonSetSection := extractResourceSection(outputStr, "kind: DaemonSet")
	assert.Contains(t, daemonSetSection, AnnotationKey+": "+testConfigHash, "DaemonSet should have config-hash annotation")
	assert.NotContains(t, daemonSetSection, "old-hash-to-be-replaced", "DaemonSet should not contain old hash")
	assert.Contains(t, daemonSetSection, "other-annotation: other-value", "DaemonSet should have preserved existing annotations")

	// Check that the ReplicaSet does not have the config-hash annotation
	replicaSetSection := extractResourceSection(outputStr, "kind: ReplicaSet")
	assert.NotContains(t, replicaSetSection, AnnotationKey, "ReplicaSet should not have config-hash annotation")
}

func createTestDeployment() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
		},
	}
}

func createTestDaemonSet() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "DaemonSet",
			"spec": map[string]interface{}{
				"template": map[string]interface{}{
					"metadata": map[string]interface{}{
						"annotations": map[string]interface{}{
							AnnotationKey:      "old-hash-to-be-replaced",
							"other-annotation": "other-value",
						},
					},
				},
			},
		},
	}
}

func createTestReplicaSet() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "ReplicaSet",
		},
	}
}

func extractResourceSection(yamlOutput, resourceMarker string) string {
	sections := strings.Split(yamlOutput, "---")
	for _, section := range sections {
		section = strings.TrimSpace(section)
		if strings.Contains(section, resourceMarker) {
			return section
		}
	}
	return ""
}
