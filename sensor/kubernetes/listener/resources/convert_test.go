package resources

import (
	"fmt"
	"math"
	"testing"
	"time"

	timestamp "github.com/gogo/protobuf/types"
	pkgV1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/sensor/kubernetes/volumes"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestConvert(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name               string
		inputObj           interface{}
		action             pkgV1.ResourceAction
		metaFieldIndex     []int
		resourceType       string
		podLister          *mockPodLister
		expectedDeployment *pkgV1.Deployment
	}{
		{
			name: "Not top-level replica set",
			inputObj: &v1beta1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							UID:  types.UID("SomeDeploymentID"),
							Name: "SomeDeployment",
						},
					},
				},
			},
			action:             pkgV1.ResourceAction_CREATE_RESOURCE,
			metaFieldIndex:     []int{1},
			resourceType:       kubernetes.ReplicaSet,
			expectedDeployment: nil,
		},
		{
			name: "Deployment",
			inputObj: &v1beta1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					UID:       types.UID("FooID"),
					Name:      "Foo",
					Namespace: "namespace",
					Labels: map[string]string{
						"key":      "value",
						"question": "answer",
					},
					Annotations: map[string]string{
						"annotationkey1": "annotationvalue1",
						"annotationkey2": "annotationvalue2",
					},
					ResourceVersion:   "100",
					CreationTimestamp: metav1.NewTime(time.Unix(1000, 0)),
				},
				Spec: v1beta1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: make(map[string]string),
					},
					Replicas: &[]int32{15}[0],
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							ServiceAccountName: "sensor",
							ImagePullSecrets: []v1.LocalObjectReference{
								{
									Name: "pull-secret1",
								},
								{
									Name: "pull-secret2",
								},
							},
							Containers: []v1.Container{
								{
									Name:    "container1",
									Args:    []string{"lorem", "ipsum"},
									Command: []string{"hello", "world"},
									Env: []v1.EnvVar{
										{
											Name:  "envName",
											Value: "envValue",
										},
									},
									Image: "docker.io/stackrox/kafka:latest",
									Ports: []v1.ContainerPort{
										{
											Name:          "api",
											ContainerPort: 9092,
											Protocol:      "TCP",
										},
										{
											Name:          "status",
											ContainerPort: 443,
											Protocol:      "UCP",
										},
									},
									SecurityContext: &v1.SecurityContext{
										SELinuxOptions: &v1.SELinuxOptions{
											User:  "user",
											Role:  "role",
											Type:  "type",
											Level: "level",
										},
									},
									VolumeMounts: []v1.VolumeMount{
										{
											Name:      "secretVol1",
											MountPath: "/var/secrets",
											ReadOnly:  true,
										},
									},
									Resources: v1.ResourceRequirements{
										Requests: v1.ResourceList{
											v1.ResourceCPU:    resource.MustParse("100m"),
											v1.ResourceMemory: resource.MustParse("1Gi"),
										},
										Limits: v1.ResourceList{
											v1.ResourceCPU:    resource.MustParse("200m"),
											v1.ResourceMemory: resource.MustParse("2Gi"),
										},
									},
								},
								{
									Name: "container2",
									Args: []string{"--flag"},
									Env: []v1.EnvVar{
										{
											Name:  "ROX_ENV_VAR",
											Value: "rox",
										},
										{
											Name:  "ROX_VERSION",
											Value: "1.0",
										},
									},
									Image: "docker.io/stackrox/policy-engine:1.3",
									SecurityContext: &v1.SecurityContext{
										Privileged: &[]bool{true}[0],
										RunAsUser:  &[]int64{0}[0],
										Capabilities: &v1.Capabilities{
											Add: []v1.Capability{
												v1.Capability("IPC_LOCK"),
												v1.Capability("SYS_RESOURCE"),
											},
										},
									},
									VolumeMounts: []v1.VolumeMount{
										{
											Name:      "hostMountVol1",
											MountPath: "/var/run/docker.sock",
										},
									},
								},
							},
							Volumes: []v1.Volume{
								{
									Name: "secretVol1",
									VolumeSource: v1.VolumeSource{
										Secret: &v1.SecretVolumeSource{
											SecretName: "private_key",
										},
									},
								},
								{
									Name: "hostMountVol1",
									VolumeSource: v1.VolumeSource{
										HostPath: &v1.HostPathVolumeSource{
											Path: "/var/run/docker.sock",
										},
									},
								},
							},
						},
					},
				},
			},
			action:         pkgV1.ResourceAction_PREEXISTING_RESOURCE,
			metaFieldIndex: []int{1},
			resourceType:   kubernetes.Deployment,
			podLister: &mockPodLister{
				pods: []v1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							UID:       types.UID("ebf487f0-a7c3-11e8-8600-42010a8a0066"),
							Name:      "mypod",
							Namespace: "myns",
						},
						Status: v1.PodStatus{
							ContainerStatuses: []v1.ContainerStatus{
								{
									Image:       "docker.io/stackrox/policy-engine:1.3",
									ImageID:     "docker-pullable://docker.io/stackrox/policy-engine@sha256:6b561c3bb9fed1b028520cce3852e6c9a6a91161df9b92ca0c3a20ebecc0581a",
									ContainerID: "docker://35669191c32a9cfb532e5d79b09f2b0926c0faf27e7543f1fbe433bd94ae78d7",
								},
							},
						},
						Spec: v1.PodSpec{
							NodeName: "mynode",
						},
					},
				},
			},
			expectedDeployment: &pkgV1.Deployment{
				Id:               "FooID",
				Name:             "Foo",
				Namespace:        "namespace",
				Type:             kubernetes.Deployment,
				Version:          "100",
				Replicas:         15,
				ServiceAccount:   "sensor",
				ImagePullSecrets: []string{"pull-secret1", "pull-secret2"},
				Labels: []*pkgV1.Deployment_KeyValue{
					{
						Key:   "key",
						Value: "value",
					},
					{
						Key:   "question",
						Value: "answer",
					},
				},
				Annotations: []*pkgV1.Deployment_KeyValue{
					{
						Key:   "annotationkey1",
						Value: "annotationvalue1",
					},
					{
						Key:   "annotationkey2",
						Value: "annotationvalue2",
					},
				},
				UpdatedAt: &timestamp.Timestamp{Seconds: 1000},
				Containers: []*pkgV1.Container{
					{
						Id: "FooID:container1",
						Config: &pkgV1.ContainerConfig{
							Command: []string{"hello", "world"},
							Args:    []string{"lorem", "ipsum"},
							Env: []*pkgV1.ContainerConfig_EnvironmentConfig{
								{
									Key:   "envName",
									Value: "envValue",
								},
							},
						},
						Image: &pkgV1.Image{
							Name: &pkgV1.ImageName{
								Registry: "docker.io",
								Remote:   "stackrox/kafka",
								Tag:      "latest",
								FullName: "docker.io/stackrox/kafka:latest",
							},
						},
						Ports: []*pkgV1.PortConfig{
							{
								Name:          "api",
								ContainerPort: 9092,
								ExposedPort:   9092,
								Protocol:      "TCP",
								Exposure:      pkgV1.PortConfig_INTERNAL,
							},
							{
								Name:          "status",
								ContainerPort: 443,
								ExposedPort:   443,
								Protocol:      "UCP",
								Exposure:      pkgV1.PortConfig_INTERNAL,
							},
						},
						Secrets: []*pkgV1.EmbeddedSecret{
							{
								Id:   "private_key",
								Name: "private_key",
								Path: "/var/secrets",
							},
						},
						SecurityContext: &pkgV1.SecurityContext{
							Selinux: &pkgV1.SecurityContext_SELinux{
								User:  "user",
								Role:  "role",
								Type:  "type",
								Level: "level",
							},
						},
						Resources: &pkgV1.Resources{
							CpuCoresRequest: 0.1,
							CpuCoresLimit:   0.2,
							MemoryMbRequest: 1024.00,
							MemoryMbLimit:   2048.00,
						},
						Instances: []*pkgV1.ContainerInstance{
							{
								InstanceId: &pkgV1.ContainerInstanceID{
									ContainerRuntime: pkgV1.ContainerRuntime_DOCKER_CONTAINER_RUNTIME,
									Id:               "35669191c32a9cfb532e5d79b09f2b0926c0faf27e7543f1fbe433bd94ae78d7",
									Node:             "mynode",
								},
								ContainingPodId: "mypod.myns@ebf487f0-a7c3-11e8-8600-42010a8a0066",
							},
						},
					},
					{
						Id: "FooID:container2",
						Config: &pkgV1.ContainerConfig{
							Args: []string{"--flag"},
							Env: []*pkgV1.ContainerConfig_EnvironmentConfig{
								{
									Key:   "ROX_ENV_VAR",
									Value: "rox",
								},
								{
									Key:   "ROX_VERSION",
									Value: "1.0",
								},
							},
							Uid: 0,
						},
						Image: &pkgV1.Image{
							Name: &pkgV1.ImageName{
								Registry: "docker.io",
								Remote:   "stackrox/policy-engine",
								Tag:      "1.3",
								Sha:      "sha256:6b561c3bb9fed1b028520cce3852e6c9a6a91161df9b92ca0c3a20ebecc0581a",
								FullName: "docker.io/stackrox/policy-engine:1.3",
							},
						},
						SecurityContext: &pkgV1.SecurityContext{
							Privileged:      true,
							AddCapabilities: []string{"IPC_LOCK", "SYS_RESOURCE"},
						},
						Volumes: []*pkgV1.Volume{

							{
								Name:        "hostMountVol1",
								Source:      "/var/run/docker.sock",
								Destination: "/var/run/docker.sock",
								Type:        "HostPath",
							},
						},
						Resources: &pkgV1.Resources{},
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual := newDeploymentEventFromResource(c.inputObj, c.action, c.metaFieldIndex, c.resourceType, c.podLister)
			assert.Equal(t, c.expectedDeployment, actual)
		})
	}
}

func verifyFloat(t *testing.T, f string, float float32) {
	floatStr := fmt.Sprintf("%0.2f", float)
	assert.Equal(t, f, floatStr)
}

func TestConvertQuantityToCores(t *testing.T) {
	cases := []struct {
		quantity resource.Quantity
		expected float32
	}{
		{
			quantity: resource.MustParse("20m"),
			expected: 0.02,
		},
		{
			quantity: resource.MustParse("200m"),
			expected: 0.2,
		},
		{
			quantity: resource.MustParse("2"),
			expected: 2.0,
		},
	}

	for _, c := range cases {
		t.Run(c.quantity.String(), func(t *testing.T) {
			assert.Equal(t, c.expected, convertQuantityToCores(&c.quantity))
		})
	}
}

func TestConvertQuantityToMb(t *testing.T) {
	cases := []struct {
		quantity resource.Quantity
		expected float32
	}{
		{
			quantity: resource.MustParse("128974848"),
			expected: 123,
		},
		{
			quantity: resource.MustParse("129e6"),
			expected: 123,
		},
		{
			quantity: resource.MustParse("129M"),
			expected: 123,
		},
		{
			quantity: resource.MustParse("123Mi"),
			expected: 123,
		},
	}

	for _, c := range cases {
		t.Run(c.quantity.String(), func(t *testing.T) {
			assert.True(t, math.Abs(float64(c.expected-convertQuantityToMb(&c.quantity))) < 0.1)
		})
	}
}

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
	w := &wrap{}
	assert.Equal(t, expectedMap, w.getVolumeSourceMap(spec))
}

type mockPodLister struct {
	pods []v1.Pod
}

func (l *mockPodLister) List(map[string]string) []v1.Pod {
	return l.pods
}
