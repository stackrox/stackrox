package resources

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/protoconv/resources/volumes"
	"github.com/stretchr/testify/assert"
	appsV1 "k8s.io/api/apps/v1"
	appsV1beta2 "k8s.io/api/apps/v1beta2"
	batchV1 "k8s.io/api/batch/v1"
	batchV1beta1 "k8s.io/api/batch/v1beta1"
	v1 "k8s.io/api/core/v1"
	extV1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	daemonSet3 := &appsV1.DaemonSet{
		Status: appsV1.DaemonSetStatus{
			NumberAvailable: 3,
		},
	}
	deploymentWrap.populateReplicas(reflect.Value{}, daemonSet3)
	assert.Equal(t, int(deploymentWrap.Replicas), 3)

	daemonSet4 := &appsV1.DaemonSet{
		Status: appsV1.DaemonSetStatus{},
	}
	deploymentWrap.populateReplicas(reflect.Value{}, daemonSet4)
	assert.Equal(t, int(deploymentWrap.Replicas), 0)
}

func TestCronJobPopulateSpec(t *testing.T) {
	deploymentWrap := &DeploymentWrap{
		Deployment: &storage.Deployment{
			Type: kubernetes.CronJob,
		},
	}

	cronJob1 := &batchV1.CronJob{
		Spec: batchV1.CronJobSpec{
			JobTemplate: batchV1.JobTemplateSpec{
				Spec: batchV1.JobSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{Containers: []v1.Container{{Name: "container1"}}},
					},
				},
			},
		},
	}
	deploymentWrap.populateFields(cronJob1)
	assert.Equal(t, deploymentWrap.Containers[0].Name, "container1")

	cronJob2 := &batchV1beta1.CronJob{
		Spec: batchV1beta1.CronJobSpec{
			JobTemplate: batchV1beta1.JobTemplateSpec{
				Spec: batchV1.JobSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{Containers: []v1.Container{{Name: "container2"}}},
					},
				},
			},
		},
	}
	deploymentWrap.populateFields(cronJob2)
	assert.Equal(t, deploymentWrap.Containers[0].Name, "container2")
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
			assert.Equal(t, IsTrackedOwnerReference(c.ref), c.isTracked)
		})
	}
}

func TestContainerLivenessProbePopulation(t *testing.T) {
	for _, testCase := range []struct {
		caseName             string
		livenessProbeDefined bool
		probe                *v1.Probe
	}{
		{
			caseName:             "Liveness probe defined.",
			livenessProbeDefined: true,
			probe:                &v1.Probe{TimeoutSeconds: 10},
		},
		{
			caseName:             "Liveness probe zero value.",
			livenessProbeDefined: false,
			probe:                &v1.Probe{},
		},
		{
			caseName:             "No liveness probe defined.",
			livenessProbeDefined: false,
			probe:                nil,
		},
	} {
		t.Run(testCase.caseName, func(t *testing.T) {
			emptyContainer := &storage.Container{}
			containers := []*storage.Container{emptyContainer}
			deploymentWrap := &DeploymentWrap{Deployment: &storage.Deployment{Containers: containers}}
			spec := v1.PodSpec{Containers: []v1.Container{{LivenessProbe: testCase.probe}}}

			deploymentWrap.populateProbes(spec)

			livenessProbe := deploymentWrap.GetContainers()[0].GetLivenessProbe()
			assert.Equal(t, livenessProbe.Defined, testCase.livenessProbeDefined)
		})
	}
}

func TestContainerLivenessProbeFromJSON(t *testing.T) {
	for _, testCase := range []struct {
		caseName             string
		livenessProbeDefined bool
		probeJSON            string
	}{
		{
			caseName:             "Readiness probe defined.",
			livenessProbeDefined: true,
			probeJSON:            `{"exec":{"command":["cat","/tmp/healthy"]}}`,
		},
		{
			caseName:             "Readiness probe not defined.",
			livenessProbeDefined: false,
			probeJSON:            `{}`,
		},
	} {
		t.Run(testCase.caseName, func(t *testing.T) {
			emptyContainer := &storage.Container{}
			containers := []*storage.Container{emptyContainer}
			deploymentWrap := &DeploymentWrap{Deployment: &storage.Deployment{Containers: containers}}

			var probe v1.Probe
			err := json.Unmarshal([]byte(testCase.probeJSON), &probe)
			spec := v1.PodSpec{Containers: []v1.Container{{LivenessProbe: &probe}}}
			deploymentWrap.populateProbes(spec)

			assert.NoError(t, err)
			livenessProbe := deploymentWrap.GetContainers()[0].GetLivenessProbe()
			assert.Equal(t, livenessProbe.Defined, testCase.livenessProbeDefined)
		})
	}
}

func TestContainerReadinessProbePopulation(t *testing.T) {
	for _, testCase := range []struct {
		caseName              string
		readinessProbeDefined bool
		probe                 *v1.Probe
	}{
		{
			caseName:              "Readiness probe defined.",
			readinessProbeDefined: true,
			probe:                 &v1.Probe{TimeoutSeconds: 10},
		},
		{
			caseName:              "Readiness probe zero value.",
			readinessProbeDefined: false,
			probe:                 &v1.Probe{},
		},
		{
			caseName:              "No readiness probe defined.",
			readinessProbeDefined: false,
			probe:                 nil,
		},
	} {
		t.Run(testCase.caseName, func(t *testing.T) {
			emptyContainer := &storage.Container{}
			containers := []*storage.Container{emptyContainer}
			deploymentWrap := &DeploymentWrap{Deployment: &storage.Deployment{Containers: containers}}
			spec := v1.PodSpec{Containers: []v1.Container{{ReadinessProbe: testCase.probe}}}

			deploymentWrap.populateProbes(spec)

			readinessProbe := deploymentWrap.GetContainers()[0].GetReadinessProbe()
			assert.Equal(t, readinessProbe.Defined, testCase.readinessProbeDefined)
		})
	}
}

func TestContainerReadinessProbeFromJSON(t *testing.T) {
	for _, testCase := range []struct {
		caseName              string
		readinessProbeDefined bool
		probeJSON             string
	}{
		{
			caseName:              "Readiness probe defined.",
			readinessProbeDefined: true,
			probeJSON:             `{"exec":{"command":["cat","/tmp/healthy"]}}`,
		},
		{
			caseName:              "Readiness probe not defined.",
			readinessProbeDefined: false,
			probeJSON:             `{}`,
		},
	} {
		t.Run(testCase.caseName, func(t *testing.T) {
			emptyContainer := &storage.Container{}
			containers := []*storage.Container{emptyContainer}
			deploymentWrap := &DeploymentWrap{Deployment: &storage.Deployment{Containers: containers}}

			var probe v1.Probe
			err := json.Unmarshal([]byte(testCase.probeJSON), &probe)
			spec := v1.PodSpec{Containers: []v1.Container{{ReadinessProbe: &probe}}}
			deploymentWrap.populateProbes(spec)

			assert.NoError(t, err)
			readinessProbe := deploymentWrap.GetContainers()[0].GetReadinessProbe()
			assert.Equal(t, readinessProbe.Defined, testCase.readinessProbeDefined)
		})
	}
}
