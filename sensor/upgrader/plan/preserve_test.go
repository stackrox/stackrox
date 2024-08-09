package plan

import (
	"testing"

	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/sensor/upgrader/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

	newDSUnstructured, err := toUnstructuredObject(newDS)
	require.NoError(t, err)
	oldDSUnstructured, err := toUnstructuredObject(oldDS)
	require.NoError(t, err)
	err = applyPreservedProperties(newDSUnstructured, oldDSUnstructured)
	require.NoError(t, err)

	var mergedDS v1.DaemonSet
	require.NoError(t, convert(scheme.Scheme, newDSUnstructured, &mergedDS))

	assert.Equal(t, expectedMergedDS, &mergedDS)
}

func TestPreserveTolerations(t *testing.T) {
	oldDeploy := &v1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sensor",
			Namespace: "stackrox",
		},
		Spec: v1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "sensor",
							Image: "foo",
						},
					},
					Tolerations: []corev1.Toleration{
						{
							Effect:   corev1.TaintEffectNoSchedule,
							Key:      "node-role.kubernetes.io/master",
							Operator: corev1.TolerationOpExists,
						},
					},
				},
			},
		},
	}

	newDeploy := &v1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sensor",
			Namespace: "stackrox",
		},
		Spec: v1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "sensor",
							Image: "bar",
						},
					},
				},
			},
		},
	}

	expectedMergedDeploy := &v1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sensor",
			Namespace: "stackrox",
		},
		Spec: v1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "sensor",
							Image: "bar",
						},
					},
					Tolerations: []corev1.Toleration{
						{
							Effect:   corev1.TaintEffectNoSchedule,
							Key:      "node-role.kubernetes.io/master",
							Operator: corev1.TolerationOpExists,
						},
					},
				},
			},
		},
	}

	newDeployUnstructured, err := toUnstructuredObject(newDeploy)
	require.NoError(t, err)
	oldDeployUnstructured, err := toUnstructuredObject(oldDeploy)
	require.NoError(t, err)
	err = applyPreservedProperties(newDeployUnstructured, oldDeployUnstructured)
	require.NoError(t, err)

	var mergedDeploy v1.Deployment
	require.NoError(t, convert(scheme.Scheme, newDeployUnstructured, &mergedDeploy))

	assert.Equal(t, expectedMergedDeploy, &mergedDeploy)
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

	oldObjUnstructured, err := toUnstructuredObject(oldObj)
	require.NoError(t, err)
	newObjUnstructured, err := toUnstructuredObject(newObj)
	require.NoError(t, err)

	err = applyPreservedProperties(newObjUnstructured, oldObjUnstructured)
	require.NoError(t, err)

	var rSvc corev1.Service
	require.NoError(t, convert(scheme.Scheme, newObjUnstructured, &rSvc))
	assert.Equal(t, "1.2.3.4", rSvc.Spec.ClusterIP)
}

// convert converts objects, adequately transferring type metadata.
func convert(scheme *runtime.Scheme, oldObj k8sutil.Object, newObj k8sutil.Object) error {
	if err := scheme.Convert(oldObj, newObj, nil); err != nil {
		return err
	}
	if newObj.GetObjectKind().GroupVersionKind() == (schema.GroupVersionKind{}) {
		newObj.GetObjectKind().SetGroupVersionKind(oldObj.GetObjectKind().GroupVersionKind())
	}
	return nil
}

func toUnstructuredObject(typedObj k8sutil.Object) (*unstructured.Unstructured, error) {
	objData, err := runtime.DefaultUnstructuredConverter.ToUnstructured(typedObj)
	if err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: objData}, nil
}
