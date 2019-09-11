package resources

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/protoconv/resources/volumes"
	"github.com/stretchr/testify/assert"
	v12 "k8s.io/api/apps/v1"
	appsV1beta2 "k8s.io/api/apps/v1beta2"
	v1 "k8s.io/api/core/v1"
	extV1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/apps"
)

func TestGetVolumeSourceMap(t *testing.T) {
	t.Parallel()

	secretVol := v1.Volume{
		Name: "secret",
		VolumeSource: v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{
				SecretName: "private_key",
			},
		},
	}
	hostPathVol := v1.Volume{
		Name: "host",
		VolumeSource: v1.VolumeSource{
			HostPath: &v1.HostPathVolumeSource{
				Path: "/var/run/docker.sock",
			},
		},
	}
	ebsVol := v1.Volume{
		Name: "ebs",
		VolumeSource: v1.VolumeSource{
			AWSElasticBlockStore: &v1.AWSElasticBlockStoreVolumeSource{
				VolumeID: "ebsVolumeID",
			},
		},
	}
	unimplementedVol := v1.Volume{
		Name: "unimplemented",
		VolumeSource: v1.VolumeSource{
			Flocker: &v1.FlockerVolumeSource{},
		},
	}

	spec := v1.PodSpec{
		Volumes: []v1.Volume{secretVol, hostPathVol, ebsVol, unimplementedVol},
	}

	expectedMap := map[string]volumes.VolumeSource{
		"secret":        volumes.VolumeRegistry["Secret"](secretVol.Secret),
		"host":          volumes.VolumeRegistry["HostPath"](hostPathVol.HostPath),
		"ebs":           volumes.VolumeRegistry["AWSElasticBlockStore"](ebsVol.AWSElasticBlockStore),
		"unimplemented": &volumes.Unimplemented{},
	}
	w := &DeploymentWrap{}
	assert.Equal(t, expectedMap, w.getVolumeSourceMap(spec))
}

func TestDaemonSetReplicas(t *testing.T) {
	deploymentWrap := &DeploymentWrap{
		Deployment: &storage.Deployment{
			Type: kubernetes.DaemonSet,
		},
	}

	daemonSet1 := &extV1beta1.DaemonSet{
		Status: extV1beta1.DaemonSetStatus{
			NumberAvailable: 1,
		},
	}
	deploymentWrap.populateReplicas(reflect.Value{}, daemonSet1)
	assert.Equal(t, int(deploymentWrap.Replicas), 1)

	daemonSet2 := &appsV1beta2.DaemonSet{
		Status: appsV1beta2.DaemonSetStatus{
			NumberAvailable: 2,
		},
	}
	deploymentWrap.populateReplicas(reflect.Value{}, daemonSet2)
	assert.Equal(t, int(deploymentWrap.Replicas), 2)

	daemonSet3 := &apps.DaemonSet{
		Status: apps.DaemonSetStatus{
			NumberAvailable: 3,
		},
	}
	deploymentWrap.populateReplicas(reflect.Value{}, daemonSet3)
	assert.Equal(t, int(deploymentWrap.Replicas), 3)

	daemonSet4 := &v12.DaemonSet{
		Status: v12.DaemonSetStatus{},
	}
	deploymentWrap.populateReplicas(reflect.Value{}, daemonSet4)
	assert.Equal(t, int(deploymentWrap.Replicas), 0)
}

func TestIsTrackedReference(t *testing.T) {
	cases := []struct {
		ref       metav1.OwnerReference
		isTracked bool
	}{
		{
			ref: metav1.OwnerReference{
				APIVersion: "v1",
				Kind:       "not a resource",
			},
			isTracked: false,
		},
		{
			ref: metav1.OwnerReference{
				APIVersion: "v1",
				Kind:       kubernetes.Deployment,
			},
			isTracked: true,
		},
		{
			ref: metav1.OwnerReference{
				APIVersion: "policy/v1beta1",
				Kind:       kubernetes.Deployment,
			},
			isTracked: true,
		},
		{
			ref: metav1.OwnerReference{
				APIVersion: "rbac.authorization.k8s.io/v1",
				Kind:       kubernetes.Deployment,
			},
			isTracked: true,
		},
		{
			ref: metav1.OwnerReference{
				APIVersion: "serving.knative.dev/v1alpha1",
				Kind:       kubernetes.Deployment,
			},
			isTracked: false,
		},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%s-%s", c.ref.APIVersion, c.ref.Kind), func(t *testing.T) {
			assert.Equal(t, isTrackedOwnerReference(c.ref), c.isTracked)
		})
	}
}
