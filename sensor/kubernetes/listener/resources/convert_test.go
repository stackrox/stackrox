package resources

import (
	"testing"
	"time"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	v1listers "k8s.io/client-go/listers/core/v1"
)

const (
	testClusterID = "12b1af66-be55-4e54-948d-ac9c311ca4b2"
)

var (
	mockNamespaceStore = func() *namespaceStore {
		s := newNamespaceStore()
		s.addNamespace(&storage.NamespaceMetadata{Id: "FAKENSID", Name: "namespace"})
		s.addNamespace(&storage.NamespaceMetadata{Id: "KUBESYSID", Name: "kube-system"})
		return s
	}()
)

func hierarchyFromPodLister(l *mockPodLister) references.ParentHierarchy {
	ph := references.NewParentHierarchy()
	if l == nil {
		return ph
	}
	pods, _ := l.List(nil)
	for _, p := range pods {
		if len(p.OwnerReferences) > 0 {
			ph.Add(p)
		}
	}
	return ph
}

func TestPopulateNonStaticFieldWithPod(t *testing.T) {
	cases := []struct {
		inputObj       interface{}
		action         central.ResourceAction
		expectedAction central.ResourceAction
	}{
		{
			inputObj: &v1.Pod{
				Status: v1.PodStatus{
					Phase: v1.PodPending,
				},
			},
			action:         central.ResourceAction_CREATE_RESOURCE,
			expectedAction: central.ResourceAction_CREATE_RESOURCE,
		},
		{
			inputObj: &v1.Pod{
				Status: v1.PodStatus{
					Phase: v1.PodFailed,
				},
			},
			action:         central.ResourceAction_CREATE_RESOURCE,
			expectedAction: central.ResourceAction_REMOVE_RESOURCE,
		},
		{
			inputObj: &v1.Pod{
				Status: v1.PodStatus{
					Phase: v1.PodSucceeded,
				},
			},
			action:         central.ResourceAction_CREATE_RESOURCE,
			expectedAction: central.ResourceAction_REMOVE_RESOURCE,
		},
	}
	storeProvider := InitializeStore()
	for _, c := range cases {
		c := c
		ph := references.NewParentHierarchy()
		newDeploymentEventFromResource(c.inputObj, &c.action, "Pod", testClusterID, nil,
			mockNamespaceStore, ph, "", storeProvider.orchestratorNamespaces)
		assert.Equal(t, c.expectedAction, c.action)
	}
}

func TestPopulateImageMetadata(t *testing.T) {
	type wrapContainer struct {
		image string
	}

	type pod struct {
		images           []string
		imageIDsInStatus []string
	}

	type metadata struct {
		expectedID             string
		expectedNotPullable    bool
		expectedIsClusterLocal bool
	}

	cases := []struct {
		name             string
		wrap             []wrapContainer
		pods             []pod
		expectedMetadata []metadata
		isClusterLocal   bool
	}{
		{
			name: "Image with latest tag, ID in status",
			wrap: []wrapContainer{
				{
					image: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/imagedigestexporter@sha256:e980d7ae539ba63dfbc19cc2ab3bc5cede348ee060e91f4d990de9352eb92c85",
				},
			},
			pods: []pod{
				{
					images: []string{"stackrox.io/main:latest"},
				},
			},
			expectedMetadata: []metadata{
				{
					expectedID: "sha256:e980d7ae539ba63dfbc19cc2ab3bc5cede348ee060e91f4d990de9352eb92c85",
				},
			},
		},
		{
			name: "Image with latest tag without ID",
			wrap: []wrapContainer{
				{
					image: "stackrox.io/main:latest",
				},
			},
			pods: []pod{
				{
					images: []string{"stackrox.io/main:latest"},
				},
			},
			expectedMetadata: []metadata{
				{
					expectedID: "",
				},
			},
		},
		{
			name: "Explicitly pullable image with latest tag, ID in status",
			wrap: []wrapContainer{
				{
					image: "stackrox.io/main:latest",
				},
			},
			pods: []pod{
				{
					images: []string{"stackrox.io/main:latest"},
					imageIDsInStatus: []string{
						"docker-pullable://stackrox.io/main@sha256:88c7e66e637f46e6bc0b95ddb1e755d616d9d76568b89af7c75c4b4aa7cfa4e3",
					},
				},
			},
			expectedMetadata: []metadata{
				{
					expectedID: "sha256:88c7e66e637f46e6bc0b95ddb1e755d616d9d76568b89af7c75c4b4aa7cfa4e3",
				},
			},
		},
		{
			name: "Image with latest tag, ID in status, not pullable",
			wrap: []wrapContainer{
				{
					image: "stackrox.io/main:latest",
				},
			},
			pods: []pod{
				{
					images: []string{"stackrox.io/main:latest"},
					imageIDsInStatus: []string{
						"docker://stackrox.io/main@sha256:88c7e66e637f46e6bc0b95ddb1e755d616d9d76568b89af7c75c4b4aa7cfa4e3",
					},
				},
			},
			expectedMetadata: []metadata{
				{
					expectedID:          "sha256:88c7e66e637f46e6bc0b95ddb1e755d616d9d76568b89af7c75c4b4aa7cfa4e3",
					expectedNotPullable: true,
				},
			},
		},
		{
			name: "Explicitly pullable image with latest tag, ID in status, two pods",
			wrap: []wrapContainer{
				{
					image: "stackrox.io/main:latest",
				},
			},
			pods: []pod{
				{
					images: []string{"stackrox.io/main:latest"},
				},
				{
					images: []string{"stackrox.io/main:latest"},
					imageIDsInStatus: []string{
						"docker-pullable://stackrox.io/main@sha256:88c7e66e637f46e6bc0b95ddb1e755d616d9d76568b89af7c75c4b4aa7cfa4e3",
					},
				},
			},
			expectedMetadata: []metadata{
				{
					expectedID: "sha256:88c7e66e637f46e6bc0b95ddb1e755d616d9d76568b89af7c75c4b4aa7cfa4e3",
				},
			},
		},
		{
			name: "Image and status mismatch",
			wrap: []wrapContainer{
				{
					image: "stackrox.io/main:latest",
				},
			},
			pods: []pod{
				{
					images: []string{"stackrox.io/main:notamatch"},
				},
				{
					images: []string{"stackrox.io/main:notamatch"},
					imageIDsInStatus: []string{
						"docker-pullable://stackrox.io/main@sha256:88c7e66e637f46e6bc0b95ddb1e755d616d9d76568b89af7c75c4b4aa7cfa4e3",
					},
				},
			},
			expectedMetadata: []metadata{
				{
					expectedID: "",
				},
			},
		},
		{
			name: "Two images, one mismatch",
			wrap: []wrapContainer{
				{
					image: "stackrox.io/main:latest",
				},
				{
					image: "stackrox.io/monitoring:latest",
				},
			},
			pods: []pod{
				{
					images: []string{
						"stackrox.io/main:notamatch",
						"stackrox.io/monitoring:latest",
					},
				},
				{
					images: []string{
						"stackrox.io/main:notamatch",
						"stackrox.io/monitoring:latest",
					},
					imageIDsInStatus: []string{
						"docker-pullable://stackrox.io/main@sha256:88c7e66e637f46e6bc0b95ddb1e755d616d9d76568b89af7c75c4b4aa7cfa4e3",
						"docker-pullable://stackrox.io/monitoring@sha256:latestformonitoring",
					},
				},
			},
			expectedMetadata: []metadata{
				{
					expectedID: "",
				},
				{
					expectedID: "sha256:latestformonitoring",
				},
			},
		},
		{
			name: "Cluster-local image with tag specified",
			wrap: []wrapContainer{
				{
					image: "image-registry.openshift-image-registry.svc:5000/testdev/nodejs-basic@sha256:31734e0a1e52996cde63a67c18eafebeb99149d0b3d0b56e1c5f31b0583ec44b",
				},
			},
			pods: []pod{
				{
					images: []string{
						"image-registry.openshift-image-registry.svc:5000/testdev/nodejs-basic@sha256:31734e0a1e52996cde63a67c18eafebeb99149d0b3d0b56e1c5f31b0583ec44b",
					},
					imageIDsInStatus: []string{
						"image-registry.openshift-image-registry.svc:5000/testdev/nodejs-basic@sha256:31734e0a1e52996cde63a67c18eafebeb99149d0b3d0b56e1c5f31b0583ec44b",
					},
				},
			},
			expectedMetadata: []metadata{
				{
					expectedID:             "sha256:31734e0a1e52996cde63a67c18eafebeb99149d0b3d0b56e1c5f31b0583ec44b",
					expectedIsClusterLocal: true,
				},
			},
			isClusterLocal: true,
		},
		{
			name: "Cluster-local image with latest tag, ID in status",
			wrap: []wrapContainer{
				{
					image: "image-registry.openshift-image-registry.svc:5000/testdev/nginx:1.18.0",
				},
			},
			pods: []pod{
				{
					images: []string{"image-registry.openshift-image-registry.svc:5000/testdev/nginx:1.18.0"},
					imageIDsInStatus: []string{
						"crio://image-registry.openshift-image-registry.svc:5000/testdev/nginx:1.18.0@sha256:e90ac5331fe095cea01b121a3627174b2e33e06e83720e9a934c7b8ccc9c55a0",
					},
				},
			},
			expectedMetadata: []metadata{
				{
					expectedID:             "sha256:e90ac5331fe095cea01b121a3627174b2e33e06e83720e9a934c7b8ccc9c55a0",
					expectedIsClusterLocal: true,
				},
			},
			isClusterLocal: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var localImages set.StringSet
			if c.isClusterLocal {
				for _, p := range c.pods {
					for _, img := range p.images {
						localImages.Add(img)
					}
				}
			}

			wrap := deploymentWrap{
				Deployment: &storage.Deployment{},
			}
			for _, container := range c.wrap {
				img, err := imageUtils.GenerateImageFromString(container.image)
				require.NoError(t, err)
				wrap.Containers = append(wrap.Containers, &storage.Container{
					Image: img,
				})

			}

			pods := make([]*v1.Pod, 0, len(c.pods))
			for _, pod := range c.pods {
				k8sPod := &v1.Pod{}
				for _, img := range pod.images {
					k8sPod.Spec.Containers = append(k8sPod.Spec.Containers, v1.Container{Image: img})
				}
				for _, imageID := range pod.imageIDsInStatus {
					k8sPod.Status.ContainerStatuses = append(k8sPod.Status.ContainerStatuses, v1.ContainerStatus{
						ImageID: imageID,
					})
				}
				pods = append(pods, k8sPod)
			}

			wrap.populateImageMetadata(localImages, pods...)
			for i, m := range c.expectedMetadata {
				assert.Equal(t, m.expectedID, wrap.Deployment.Containers[i].Image.Id)
				assert.Equal(t, m.expectedNotPullable, wrap.Deployment.Containers[i].Image.NotPullable)
				assert.Equal(t, m.expectedIsClusterLocal, wrap.Deployment.Containers[i].Image.IsClusterLocal)
			}
		})
	}
}

func TestConvert(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name               string
		inputObj           interface{}
		deploymentType     string
		action             central.ResourceAction
		podLister          *mockPodLister
		expectedDeployment *storage.Deployment
	}{
		{
			name: "Not top-level replica set",
			inputObj: &v1beta1.ReplicaSet{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1beta1",
					Kind:       "ReplicaSet",
				},
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							UID:        types.UID("SomeDeploymentID"),
							Name:       "SomeDeployment",
							APIVersion: "v1beta1/deployments",
							Kind:       "Deployment",
						},
					},
				},
			},
			deploymentType:     kubernetes.ReplicaSet,
			action:             central.ResourceAction_CREATE_RESOURCE,
			expectedDeployment: nil,
		},
		{
			name: "Deployment",
			inputObj: &v1beta1.Deployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1beta1",
					Kind:       "Deployment",
				},
				ObjectMeta: metav1.ObjectMeta{
					UID:       types.UID("FooID"),
					Name:      "deployment",
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
							ServiceAccountName:           "sensor",
							AutomountServiceAccountToken: &[]bool{true}[0],
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
										ReadOnlyRootFilesystem: &[]bool{true}[0],
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
									LivenessProbe:  &v1.Probe{TimeoutSeconds: 10},
									ReadinessProbe: &v1.Probe{TimeoutSeconds: 10},
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
										ReadOnlyRootFilesystem:   &[]bool{true}[0],
										AllowPrivilegeEscalation: &[]bool{true}[0],
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
			deploymentType: kubernetes.Deployment,
			action:         central.ResourceAction_UPDATE_RESOURCE,
			podLister: &mockPodLister{
				pods: []*v1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							UID:       types.UID("ebf487f0-a7c3-11e8-8600-42010a8a0066"),
							Name:      "deployment-blah-blah",
							Namespace: "myns",
							OwnerReferences: []metav1.OwnerReference{
								{
									UID:  "FooID",
									Kind: kubernetes.Deployment,
								},
							},
						},
						Status: v1.PodStatus{
							ContainerStatuses: []v1.ContainerStatus{
								{
									Name:    "container1",
									Image:   "docker.io/stackrox/kafka:latest",
									ImageID: "docker://docker.io/stackrox/kafka@sha256:aa561c3bb9fed1b028520cce3852e6c9a6a91161df9b92ca0c3a20ebecc0581a",
								},
								{
									Name:        "container2",
									Image:       "docker.io/stackrox/policy-engine:1.3",
									ImageID:     "docker-pullable://docker.io/stackrox/policy-engine@sha256:6b561c3bb9fed1b028520cce3852e6c9a6a91161df9b92ca0c3a20ebecc0581a",
									ContainerID: "docker://35669191c32a9cfb532e5d79b09f2b0926c0faf27e7543f1fbe433bd94ae78d7",
								},
							},
						},
						Spec: v1.PodSpec{
							NodeName:                     "mynode",
							AutomountServiceAccountToken: &[]bool{true}[0],
							Containers: []v1.Container{
								{
									Name:  "container1",
									Image: "docker.io/stackrox/kafka:latest",
								},
								{
									Name:  "container2",
									Image: "docker.io/stackrox/policy-engine:1.3",
								},
							},
						},
					},
				},
			},
			expectedDeployment: &storage.Deployment{
				Id:                           "FooID",
				ClusterId:                    testClusterID,
				Name:                         "deployment",
				Namespace:                    "namespace",
				NamespaceId:                  "FAKENSID",
				Type:                         kubernetes.Deployment,
				Replicas:                     15,
				ServiceAccount:               "sensor",
				ImagePullSecrets:             []string{"pull-secret1", "pull-secret2"},
				AutomountServiceAccountToken: true,
				Labels: map[string]string{
					"key":      "value",
					"question": "answer",
				},
				LabelSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{},
				},
				Annotations: map[string]string{
					"annotationkey1": "annotationvalue1",
					"annotationkey2": "annotationvalue2",
				},
				Created:     &timestamp.Timestamp{Seconds: 1000},
				Tolerations: []*storage.Toleration{},
				Ports: []*storage.PortConfig{
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
				Containers: []*storage.Container{
					{
						Id:   "FooID:container1",
						Name: "container1",
						Config: &storage.ContainerConfig{
							Command: []string{"hello", "world"},
							Args:    []string{"lorem", "ipsum"},
							Env: []*storage.ContainerConfig_EnvironmentConfig{
								{
									Key:          "envName",
									Value:        "envValue",
									EnvVarSource: storage.ContainerConfig_EnvironmentConfig_RAW,
								},
							},
						},
						Image: &storage.ContainerImage{
							Id: "sha256:aa561c3bb9fed1b028520cce3852e6c9a6a91161df9b92ca0c3a20ebecc0581a",
							Name: &storage.ImageName{
								Registry: "docker.io",
								Remote:   "stackrox/kafka",
								Tag:      "latest",
								FullName: "docker.io/stackrox/kafka:latest",
							},
							NotPullable: true,
						},
						Secrets: []*storage.EmbeddedSecret{
							{
								Name: "private_key",
								Path: "/var/secrets",
							},
							{
								Name: "pull-secret1",
							},
							{
								Name: "pull-secret2",
							},
						},
						SecurityContext: &storage.SecurityContext{
							Selinux: &storage.SecurityContext_SELinux{
								User:  "user",
								Role:  "role",
								Type:  "type",
								Level: "level",
							},
							ReadOnlyRootFilesystem: true,
						},
						Resources: &storage.Resources{
							CpuCoresRequest: 0.1,
							CpuCoresLimit:   0.2,
							MemoryMbRequest: 1024.00,
							MemoryMbLimit:   2048.00,
						},
						Ports: []*storage.PortConfig{
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
						LivenessProbe:  &storage.LivenessProbe{Defined: true},
						ReadinessProbe: &storage.ReadinessProbe{Defined: true},
					},
					{
						Id:   "FooID:container2",
						Name: "container2",
						Config: &storage.ContainerConfig{
							Args: []string{"--flag"},
							Env: []*storage.ContainerConfig_EnvironmentConfig{
								{
									Key:          "ROX_ENV_VAR",
									Value:        "rox",
									EnvVarSource: storage.ContainerConfig_EnvironmentConfig_RAW,
								},
								{
									Key:          "ROX_VERSION",
									Value:        "1.0",
									EnvVarSource: storage.ContainerConfig_EnvironmentConfig_RAW,
								},
							},
							Uid: 0,
						},
						Image: &storage.ContainerImage{
							Id: "sha256:6b561c3bb9fed1b028520cce3852e6c9a6a91161df9b92ca0c3a20ebecc0581a",
							Name: &storage.ImageName{
								Registry: "docker.io",
								Remote:   "stackrox/policy-engine",
								Tag:      "1.3",
								FullName: "docker.io/stackrox/policy-engine:1.3",
							},
							NotPullable: false,
						},
						Secrets: []*storage.EmbeddedSecret{
							{
								Name: "pull-secret1",
							},
							{
								Name: "pull-secret2",
							},
						},
						SecurityContext: &storage.SecurityContext{
							Privileged:               true,
							AddCapabilities:          []string{"IPC_LOCK", "SYS_RESOURCE"},
							ReadOnlyRootFilesystem:   true,
							AllowPrivilegeEscalation: true,
						},
						Volumes: []*storage.Volume{

							{
								Name:        "hostMountVol1",
								Source:      "/var/run/docker.sock",
								Destination: "/var/run/docker.sock",
								Type:        "HostPath",
							},
						},
						Resources:      &storage.Resources{},
						LivenessProbe:  &storage.LivenessProbe{Defined: false},
						ReadinessProbe: &storage.ReadinessProbe{Defined: false},
					},
				},
			},
		},
		{
			name: "CronJob",
			inputObj: &batchv1.CronJob{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "CronJob",
				},
				ObjectMeta: metav1.ObjectMeta{
					UID:       types.UID("FooID"),
					Name:      "cronjob",
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
				Spec: batchv1.CronJobSpec{
					Schedule: "* * * * *",
					JobTemplate: batchv1.JobTemplateSpec{
						Spec: batchv1.JobSpec{
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{},
							},
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{},
								Spec: v1.PodSpec{
									ServiceAccountName:           "sensor",
									AutomountServiceAccountToken: &[]bool{true}[0],
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
												ReadOnlyRootFilesystem: &[]bool{true}[0],
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
											LivenessProbe:  &v1.Probe{TimeoutSeconds: 10},
											ReadinessProbe: &v1.Probe{TimeoutSeconds: 10},
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
												ReadOnlyRootFilesystem:   &[]bool{true}[0],
												AllowPrivilegeEscalation: &[]bool{true}[0],
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
				},
			},
			deploymentType: kubernetes.CronJob,
			action:         central.ResourceAction_UPDATE_RESOURCE,
			podLister: &mockPodLister{
				pods: []*v1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							UID:       types.UID("ebf487f0-a7c3-11e8-8600-42010a8a0066"),
							Name:      "cronjob-blah-blah",
							Namespace: "myns",
							OwnerReferences: []metav1.OwnerReference{
								{
									UID:  "FooID",
									Kind: kubernetes.Deployment,
								},
							},
						},
						Status: v1.PodStatus{
							ContainerStatuses: []v1.ContainerStatus{
								{
									Name:    "container1",
									Image:   "docker.io/stackrox/kafka:latest",
									ImageID: "docker://docker.io/stackrox/kafka@sha256:aa561c3bb9fed1b028520cce3852e6c9a6a91161df9b92ca0c3a20ebecc0581a",
								},
								{
									Name:        "container2",
									Image:       "docker.io/stackrox/policy-engine:1.3",
									ImageID:     "docker-pullable://docker.io/stackrox/policy-engine@sha256:6b561c3bb9fed1b028520cce3852e6c9a6a91161df9b92ca0c3a20ebecc0581a",
									ContainerID: "docker://35669191c32a9cfb532e5d79b09f2b0926c0faf27e7543f1fbe433bd94ae78d7",
								},
							},
						},
						Spec: v1.PodSpec{
							NodeName:                     "mynode",
							AutomountServiceAccountToken: &[]bool{true}[0],
							Containers: []v1.Container{
								{
									Name:  "container1",
									Image: "docker.io/stackrox/kafka:latest",
								},
								{
									Name:  "container2",
									Image: "docker.io/stackrox/policy-engine:1.3",
								},
							},
						},
					},
				},
			},
			expectedDeployment: &storage.Deployment{
				Id:                           "FooID",
				ClusterId:                    testClusterID,
				Name:                         "cronjob",
				Namespace:                    "namespace",
				NamespaceId:                  "FAKENSID",
				Type:                         kubernetes.CronJob,
				Replicas:                     0,
				ServiceAccount:               "sensor",
				ImagePullSecrets:             []string{"pull-secret1", "pull-secret2"},
				AutomountServiceAccountToken: true,
				Labels: map[string]string{
					"key":      "value",
					"question": "answer",
				},
				LabelSelector: &storage.LabelSelector{
					MatchLabels: map[string]string{},
				},
				Annotations: map[string]string{
					"annotationkey1": "annotationvalue1",
					"annotationkey2": "annotationvalue2",
				},
				Created:     &timestamp.Timestamp{Seconds: 1000},
				Tolerations: []*storage.Toleration{},
				Ports: []*storage.PortConfig{
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
				Containers: []*storage.Container{
					{
						Id:   "FooID:container1",
						Name: "container1",
						Config: &storage.ContainerConfig{
							Command: []string{"hello", "world"},
							Args:    []string{"lorem", "ipsum"},
							Env: []*storage.ContainerConfig_EnvironmentConfig{
								{
									Key:          "envName",
									Value:        "envValue",
									EnvVarSource: storage.ContainerConfig_EnvironmentConfig_RAW,
								},
							},
						},
						Image: &storage.ContainerImage{
							Id: "sha256:aa561c3bb9fed1b028520cce3852e6c9a6a91161df9b92ca0c3a20ebecc0581a",
							Name: &storage.ImageName{
								Registry: "docker.io",
								Remote:   "stackrox/kafka",
								Tag:      "latest",
								FullName: "docker.io/stackrox/kafka:latest",
							},
							NotPullable: true,
						},
						Secrets: []*storage.EmbeddedSecret{
							{
								Name: "private_key",
								Path: "/var/secrets",
							},
							{
								Name: "pull-secret1",
							},
							{
								Name: "pull-secret2",
							},
						},
						SecurityContext: &storage.SecurityContext{
							Selinux: &storage.SecurityContext_SELinux{
								User:  "user",
								Role:  "role",
								Type:  "type",
								Level: "level",
							},
							ReadOnlyRootFilesystem: true,
						},
						Resources: &storage.Resources{
							CpuCoresRequest: 0.1,
							CpuCoresLimit:   0.2,
							MemoryMbRequest: 1024.00,
							MemoryMbLimit:   2048.00,
						},
						Ports: []*storage.PortConfig{
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
						LivenessProbe:  &storage.LivenessProbe{Defined: true},
						ReadinessProbe: &storage.ReadinessProbe{Defined: true},
					},
					{
						Id:   "FooID:container2",
						Name: "container2",
						Config: &storage.ContainerConfig{
							Args: []string{"--flag"},
							Env: []*storage.ContainerConfig_EnvironmentConfig{
								{
									Key:          "ROX_ENV_VAR",
									Value:        "rox",
									EnvVarSource: storage.ContainerConfig_EnvironmentConfig_RAW,
								},
								{
									Key:          "ROX_VERSION",
									Value:        "1.0",
									EnvVarSource: storage.ContainerConfig_EnvironmentConfig_RAW,
								},
							},
							Uid: 0,
						},
						Image: &storage.ContainerImage{
							Id: "sha256:6b561c3bb9fed1b028520cce3852e6c9a6a91161df9b92ca0c3a20ebecc0581a",
							Name: &storage.ImageName{
								Registry: "docker.io",
								Remote:   "stackrox/policy-engine",
								Tag:      "1.3",
								FullName: "docker.io/stackrox/policy-engine:1.3",
							},
							NotPullable: false,
						},
						Secrets: []*storage.EmbeddedSecret{
							{
								Name: "pull-secret1",
							},
							{
								Name: "pull-secret2",
							},
						},
						SecurityContext: &storage.SecurityContext{
							Privileged:               true,
							AddCapabilities:          []string{"IPC_LOCK", "SYS_RESOURCE"},
							ReadOnlyRootFilesystem:   true,
							AllowPrivilegeEscalation: true,
						},
						Volumes: []*storage.Volume{

							{
								Name:        "hostMountVol1",
								Source:      "/var/run/docker.sock",
								Destination: "/var/run/docker.sock",
								Type:        "HostPath",
							},
						},
						Resources:      &storage.Resources{},
						LivenessProbe:  &storage.LivenessProbe{Defined: false},
						ReadinessProbe: &storage.ReadinessProbe{Defined: false},
					},
				},
			},
		},
	}

	storeProvider := InitializeStore()
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			actual := newDeploymentEventFromResource(c.inputObj, &c.action, c.deploymentType, testClusterID,
				c.podLister, mockNamespaceStore, hierarchyFromPodLister(c.podLister), "",
				storeProvider.orchestratorNamespaces).GetDeployment()
			if actual != nil {
				actual.StateTimestamp = 0
			}
			assert.Equal(t, c.expectedDeployment, actual)
		})
	}
}

type mockPodLister struct {
	v1listers.PodLister
	v1listers.PodNamespaceLister
	pods []*v1.Pod
}

func (l *mockPodLister) List(_ labels.Selector) ([]*v1.Pod, error) {
	return l.pods, nil
}

func (l *mockPodLister) Pods(_ string) v1listers.PodNamespaceLister {
	return l
}
