package translation

import (
	"testing"

	"github.com/jeremywohl/flatten"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	testingUtils "github.com/stackrox/rox/operator/internal/values/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	fkClient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDeploymentDefaults(t *testing.T) {
	componentPaths := []testingUtils.ComponentPath{
		{Name: "central", NodeSelectorPath: "central.nodeSelector", TolerationsPath: "central.tolerations"},
		{Name: "central-db", NodeSelectorPath: "central.db.nodeSelector", TolerationsPath: "central.db.tolerations"},
		{Name: "scanner", NodeSelectorPath: "scanner.nodeSelector", TolerationsPath: "scanner.tolerations"},
		{Name: "scanner-db", NodeSelectorPath: "scanner.dbNodeSelector", TolerationsPath: "scanner.dbTolerations"},
		{Name: "scannerV4-indexer", NodeSelectorPath: "scannerV4.indexer.nodeSelector", TolerationsPath: "scannerV4.indexer.tolerations"},
		{Name: "scannerV4-matcher", NodeSelectorPath: "scannerV4.matcher.nodeSelector", TolerationsPath: "scannerV4.matcher.tolerations"},
		{Name: "scannerV4-db", NodeSelectorPath: "scannerV4.db.nodeSelector", TolerationsPath: "scannerV4.db.tolerations"},
		{Name: "configController", NodeSelectorPath: "configController.nodeSelector", TolerationsPath: "configController.tolerations"},
	}

	tests := map[string]struct {
		central      platform.Central
		expectations testingUtils.SchedulingExpectations
	}{
		"pinToNodes InfraRole": {
			central: platform.Central{
				ObjectMeta: metav1.ObjectMeta{Namespace: "stackrox"},
				Spec: platform.CentralSpec{
					Customize: &platform.CustomizeSpec{
						DeploymentDefaults: &platform.DeploymentDefaultsSpec{
							PinToNodes: ptr.To(platform.PinToNodesInfraRole),
						},
					},
				},
			},
			expectations: testingUtils.NewSchedulingExpectations(componentPaths, testingUtils.InfraScheduling),
		},
		"explicit nodeSelector and tolerations": {
			central: platform.Central{
				ObjectMeta: metav1.ObjectMeta{Namespace: "stackrox"},
				Spec: platform.CentralSpec{
					Customize: &platform.CustomizeSpec{
						DeploymentDefaults: &platform.DeploymentDefaultsSpec{
							NodeSelector: map[string]string{"global-label": "global-value"},
							Tolerations:  []*corev1.Toleration{{Key: "global-taint", Operator: corev1.TolerationOpExists}},
						},
					},
				},
			},
			expectations: testingUtils.NewSchedulingExpectations(componentPaths, testingUtils.GlobalScheduling),
		},
		"component-specific overrides global": {
			central: platform.Central{
				ObjectMeta: metav1.ObjectMeta{Namespace: "stackrox"},
				Spec: platform.CentralSpec{
					Central: &platform.CentralComponentSpec{
						DeploymentSpec: platform.DeploymentSpec{
							NodeSelector: map[string]string{"component-label": "component-value"},
							Tolerations:  []*corev1.Toleration{{Key: "component-taint", Effect: corev1.TaintEffectNoExecute}},
						},
					},
					Customize: &platform.CustomizeSpec{
						DeploymentDefaults: &platform.DeploymentDefaultsSpec{
							NodeSelector: map[string]string{"global-label": "global-value"},
							Tolerations:  []*corev1.Toleration{{Key: "global-taint", Operator: corev1.TolerationOpExists}},
						},
					},
				},
			},
			expectations: testingUtils.NewSchedulingExpectations(componentPaths, testingUtils.GlobalScheduling).
				WithOverride("central", testingUtils.SchedulingExpectation{
					NodeSelector: map[string]any{"component-label": "component-value"},
					Tolerations:  []any{map[string]any{"effect": "NoExecute", "key": "component-taint"}},
				}),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			translator := Translator{client: fkClient.NewClientBuilder().Build()}
			values, err := translator.translate(t.Context(), tt.central)
			require.NoError(t, err)

			flatValues, err := flatten.Flatten(values, "", flatten.DotStyle)
			require.NoError(t, err)

			for _, path := range componentPaths {
				t.Run(path.Name, func(t *testing.T) {
					exp := tt.expectations[path.Name]
					flatExpected, err := flatten.Flatten(map[string]any{
						path.NodeSelectorPath: exp.NodeSelector,
						path.TolerationsPath:  exp.Tolerations,
					}, "", flatten.DotStyle)
					require.NoError(t, err)

					for k, v := range flatExpected {
						assert.Equal(t, v, flatValues[k], "mismatch at %s", k)
					}
				})
			}
		})
	}
}
