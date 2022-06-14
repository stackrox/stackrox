package plan

import (
	"testing"

	"github.com/stackrox/stackrox/sensor/upgrader/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestPreserveResources(t *testing.T) {
	oldDS := &v1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "collector",
			Namespace: "stackrox",
			Annotations: map[string]string{
				common.PreserveResourcesAnnotationKey: "true",
			},
		},
		Spec: v1.DaemonSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "collector",
							Image: "foo",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("501m"),
									corev1.ResourceMemory: resource.MustParse("2Gi"),
								},
							},
						},
						{
							Name:  "compliance",
							Image: "compliancefoo",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("601m"),
									corev1.ResourceMemory: resource.MustParse("2.5Gi"),
								},
							},
						},
					},
				},
			},
		},
	}

	newDS := &v1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "collector",
			Namespace: "stackrox",
		},
		Spec: v1.DaemonSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "collector",
							Image: "bar",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:     resource.MustParse("500m"),
									corev1.ResourceStorage: resource.MustParse("10Gi"),
								},
							},
						},
						{
							Name:  "newcompliance",
							Image: "compliancebar",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:     resource.MustParse("600m"),
									corev1.ResourceStorage: resource.MustParse("20Gi"),
								},
							},
						},
					},
				},
			},
		},
	}

	expectedMergedDS := &v1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "collector",
			Namespace: "stackrox",
			Annotations: map[string]string{
				common.PreserveResourcesAnnotationKey: "true",
			},
		},
		Spec: v1.DaemonSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "collector",
							Image: "bar",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:     resource.MustParse("501m"),
									corev1.ResourceMemory:  resource.MustParse("2Gi"),
									corev1.ResourceStorage: resource.MustParse("10Gi"),
								},
							},
						},
						{
							Name:  "newcompliance",
							Image: "compliancebar",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:     resource.MustParse("600m"),
									corev1.ResourceStorage: resource.MustParse("20Gi"),
								},
							},
						},
					},
				},
			},
		},
	}

	mergedDS, err := applyPreservedProperties(scheme.Scheme, newDS, oldDS)
	require.NoError(t, err)

	assert.Equal(t, expectedMergedDS, mergedDS)
}

func Test_applyPreservedProperties(t *testing.T) {
	oldObj := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "1.2.3.4",
		},
	}

	newObj := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
	}

	r, err := applyPreservedProperties(scheme.Scheme, newObj, oldObj)
	require.NoError(t, err)
	assert.Equal(t, serviceGVK, r.GetObjectKind().GroupVersionKind())
	assert.Equal(t, "1.2.3.4", r.(*corev1.Service).Spec.ClusterIP)
}
