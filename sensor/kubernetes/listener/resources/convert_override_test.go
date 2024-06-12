package resources

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/testutils"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestConvertWithRegistryOverride(t *testing.T) {
	base := struct {
		inputObj           interface{}
		deploymentType     string
		action             central.ResourceAction
		podLister          *mockPodLister
		expectedDeployment *storage.Deployment
		registryOverride   string
	}{
		registryOverride: "hello.io",
		inputObj: &v1beta1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				UID:               types.UID("FooID"),
				Name:              "deployment",
				Namespace:         "namespace",
				CreationTimestamp: metav1.NewTime(time.Unix(1000, 0)),
			},
			Spec: v1beta1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: make(map[string]string),
				},
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name:  "container1",
								Image: "stackrox/kafka:latest",
							},
							{
								Name:  "container2",
								Image: "stackrox/policy-engine:1.3",
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
								Image:   "stackrox/kafka:latest",
								ImageID: "docker://quay.io/stackrox/kafka@sha256:aa561c3bb9fed1b028520cce3852e6c9a6a91161df9b92ca0c3a20ebecc0581a",
							},
							{
								Name:        "container2",
								Image:       "stackrox/policy-engine:1.3",
								ImageID:     "docker-pullable://quay.io/stackrox/policy-engine@sha256:6b561c3bb9fed1b028520cce3852e6c9a6a91161df9b92ca0c3a20ebecc0581a",
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
								Image: "stackrox/kafka:latest",
							},
							{
								Name:  "container2",
								Image: "stackrox/policy-engine:1.3",
							},
						},
					},
				},
			},
		},
		expectedDeployment: &storage.Deployment{
			Id:          "FooID",
			ClusterId:   testClusterID,
			Name:        "deployment",
			Namespace:   "namespace",
			NamespaceId: "FAKENSID",
			Type:        kubernetes.Deployment,
			LabelSelector: &storage.LabelSelector{
				MatchLabels: map[string]string{},
			},
			Created:                      protocompat.GetProtoTimestampFromSeconds(1000),
			Tolerations:                  []*storage.Toleration{},
			ServiceAccount:               "default",
			AutomountServiceAccountToken: true,
			ImagePullSecrets:             []string{},
			Containers: []*storage.Container{
				{
					Id:   "FooID:container1",
					Name: "container1",
					Config: &storage.ContainerConfig{
						Env: []*storage.ContainerConfig_EnvironmentConfig{},
					},
					SecurityContext: &storage.SecurityContext{},
					Resources:       &storage.Resources{},
					Image: &storage.ContainerImage{
						Id:          "sha256:aa561c3bb9fed1b028520cce3852e6c9a6a91161df9b92ca0c3a20ebecc0581a",
						Name:        &storage.ImageName{},
						NotPullable: true,
					},
					LivenessProbe:  &storage.LivenessProbe{Defined: false},
					ReadinessProbe: &storage.ReadinessProbe{Defined: false},
				},
				{
					Id:   "FooID:container2",
					Name: "container2",
					Config: &storage.ContainerConfig{
						Env: []*storage.ContainerConfig_EnvironmentConfig{},
					},
					Image: &storage.ContainerImage{
						Id:   "sha256:6b561c3bb9fed1b028520cce3852e6c9a6a91161df9b92ca0c3a20ebecc0581a",
						Name: &storage.ImageName{},
					},
					SecurityContext: &storage.SecurityContext{},
					Resources:       &storage.Resources{},
					LivenessProbe:   &storage.LivenessProbe{Defined: false},
					ReadinessProbe:  &storage.ReadinessProbe{Defined: false},
				},
			},
		},
	}

	cases := []struct {
		name                     string
		enableUnqualifiedFeature bool
		expectedImageNames       []*storage.ImageName
	}{
		{
			"use registry override",
			false,
			[]*storage.ImageName{
				{
					Registry: "hello.io",
					Remote:   "stackrox/kafka",
					Tag:      "latest",
					FullName: "hello.io/stackrox/kafka:latest",
				}, {
					Registry: "hello.io",
					Remote:   "stackrox/policy-engine",
					Tag:      "1.3",
					FullName: "hello.io/stackrox/policy-engine:1.3",
				},
			},
		},
		{
			"use registry from runtime",
			true,
			[]*storage.ImageName{
				{
					Registry: "quay.io",
					Remote:   "stackrox/kafka",
					Tag:      "latest",
					FullName: "quay.io/stackrox/kafka:latest",
				},
				{
					Registry: "quay.io",
					Remote:   "stackrox/policy-engine",
					Tag:      "1.3",
					FullName: "quay.io/stackrox/policy-engine:1.3",
				},
			},
		},
	}

	storeProvider := InitializeStore()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			testutils.MustUpdateFeature(t, features.UnqualifiedSearchRegistries, c.enableUnqualifiedFeature)
			for i, container := range base.expectedDeployment.Containers {
				container.Image.Name = c.expectedImageNames[i]
			}

			actual := newDeploymentEventFromResource(base.inputObj, &base.action, base.deploymentType, testClusterID,
				base.podLister, mockNamespaceStore, hierarchyFromPodLister(base.podLister), base.registryOverride,
				storeProvider.orchestratorNamespaces).GetDeployment()
			if actual != nil {
				actual.StateTimestamp = 0
			}
			protoassert.Equal(t, base.expectedDeployment, actual)
		})
	}
}
