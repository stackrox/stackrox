//go:build test

package testmetrics

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCollectFromPods_Validation(t *testing.T) {
	ctx := context.Background()
	cs := fake.NewSimpleClientset()
	_, err := collectFromPods(ctx, cs, ScrapeTarget{})
	require.Error(t, err)
}

func TestFindServicePort_ContinuesAcrossMatchingServices(t *testing.T) {
	ctx := context.Background()
	cs := fake.NewSimpleClientset(
		&coreV1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "aaa-collector-wrong", Namespace: "stackrox"},
			Spec: coreV1.ServiceSpec{
				Selector: map[string]string{"app": "collector"},
				Ports: []coreV1.ServicePort{{
					Name:       "wrong",
					Port:       8080,
					TargetPort: intstr.FromInt32(8080),
				}},
			},
		},
		&coreV1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "zzz-collector-right", Namespace: "stackrox"},
			Spec: coreV1.ServiceSpec{
				Selector: map[string]string{"app": "collector"},
				Ports: []coreV1.ServicePort{{
					Name:       "metrics",
					Port:       9091,
					TargetPort: intstr.FromInt32(9091),
				}},
			},
		},
	)

	err := FindServicePort(ctx, cs, "stackrox", "app", "collector", 9091)
	require.NoError(t, err)
}
