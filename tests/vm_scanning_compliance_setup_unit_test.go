//go:build test_e2e_vm

package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestSetContainerEnv(t *testing.T) {
	testCases := map[string]struct {
		envBefore string
		envAfter  string
		changed   bool
	}{
		"should update when value differs": {
			envBefore: "disabled",
			envAfter:  ":9091",
			changed:   true,
		},
		"should be no-op when value matches": {
			envBefore: ":9091",
			envAfter:  ":9091",
			changed:   false,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			ds := &appsV1.DaemonSet{
				Spec: appsV1.DaemonSetSpec{
					Template: coreV1.PodTemplateSpec{
						Spec: coreV1.PodSpec{
							Containers: []coreV1.Container{{
								Name: "compliance",
								Env:  []coreV1.EnvVar{{Name: "ROX_METRICS_PORT", Value: tc.envBefore}},
							}},
						},
					},
				},
			}
			changed, err := setContainerEnv(ds, "compliance", "ROX_METRICS_PORT", tc.envAfter)
			require.NoError(t, err)
			require.Equal(t, tc.changed, changed)
			require.Equal(t, tc.envAfter, ds.Spec.Template.Spec.Containers[0].Env[0].Value)
		})
	}
}

func TestEnsureComplianceMetricsEnv_RetriesOnConflict(t *testing.T) {
	ds := &appsV1.DaemonSet{
		ObjectMeta: metaV1.ObjectMeta{Name: "collector", Namespace: "stackrox"},
		Spec: appsV1.DaemonSetSpec{
			Template: coreV1.PodTemplateSpec{
				Spec: coreV1.PodSpec{
					Containers: []coreV1.Container{{
						Name: "compliance",
						Env:  []coreV1.EnvVar{{Name: "ROX_METRICS_PORT", Value: "disabled"}},
					}},
				},
			},
		},
		Status: appsV1.DaemonSetStatus{
			DesiredNumberScheduled: 1,
			UpdatedNumberScheduled: 1,
			NumberReady:            1,
			ObservedGeneration:     1,
		},
	}
	cs := fake.NewSimpleClientset(ds)
	updateAttempts := 0
	cs.PrependReactor("update", "daemonsets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		updateAttempts++
		if updateAttempts == 1 {
			return true, nil, apierrors.NewConflict(
				schema.GroupResource{Group: "apps", Resource: "daemonsets"},
				"collector",
				errors.New("conflict"),
			)
		}
		return false, nil, nil
	})

	s := &VMScanningSuite{k8sClient: cs}
	s.SetT(t)
	s.ctx = t.Context()

	s.ensureComplianceMetricsEnv(t.Context(), "stackrox", "collector", "compliance", "ROX_METRICS_PORT", ":9091")

	got, err := cs.AppsV1().DaemonSets("stackrox").Get(context.Background(), "collector", metaV1.GetOptions{})
	require.NoError(t, err)
	require.GreaterOrEqual(t, updateAttempts, 2)
	require.Equal(t, ":9091", got.Spec.Template.Spec.Containers[0].Env[0].Value)
}
