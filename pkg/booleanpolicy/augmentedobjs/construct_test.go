package augmentedobjs

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stretchr/testify/require"
)

func TestConstructDeploymentIncludesInitContainers(t *testing.T) {
	t.Setenv(features.InitContainerSupport.EnvVar(), "true")

	deployment := &storage.Deployment{
		Name:      "test-deploy",
		Namespace: "default",
		Containers: []*storage.Container{
			{Name: "init", Type: storage.ContainerType_INIT},
			{Name: "main", Type: storage.ContainerType_REGULAR},
		},
	}
	images := []*storage.Image{
		{Id: "init-image"},
		{Id: "main-image"},
	}

	obj, err := ConstructDeployment(deployment, images, &NetworkPoliciesApplied{})
	require.NoError(t, err)
	require.NotNil(t, obj)
}

func TestConstructDeploymentWithProcessIncludesInitContainers(t *testing.T) {
	cases := map[string]struct {
		containers       []*storage.Container
		images           []*storage.Image
		processContainer string
	}{
		"process on regular container with init present": {
			containers: []*storage.Container{
				{Name: "init", Type: storage.ContainerType_INIT},
				{Name: "main", Type: storage.ContainerType_REGULAR},
			},
			images:           []*storage.Image{{Id: "init-img"}, {Id: "main-img"}},
			processContainer: "main",
		},
		"process on init container": {
			containers: []*storage.Container{
				{Name: "init", Type: storage.ContainerType_INIT},
				{Name: "main", Type: storage.ContainerType_REGULAR},
			},
			images:           []*storage.Image{{Id: "init-img"}, {Id: "main-img"}},
			processContainer: "init",
		},
		"multiple init and regular containers": {
			containers: []*storage.Container{
				{Name: "init-db", Type: storage.ContainerType_INIT},
				{Name: "init-config", Type: storage.ContainerType_INIT},
				{Name: "api", Type: storage.ContainerType_REGULAR},
				{Name: "worker", Type: storage.ContainerType_REGULAR},
				{Name: "sidecar", Type: storage.ContainerType_REGULAR},
			},
			images:           []*storage.Image{{Id: "init-db-img"}, {Id: "init-config-img"}, {Id: "api-img"}, {Id: "worker-img"}, {Id: "sidecar-img"}},
			processContainer: "sidecar",
		},
		"no init containers": {
			containers: []*storage.Container{
				{Name: "app", Type: storage.ContainerType_REGULAR},
				{Name: "sidecar", Type: storage.ContainerType_REGULAR},
			},
			images:           []*storage.Image{{Id: "app-img"}, {Id: "sidecar-img"}},
			processContainer: "sidecar",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv(features.InitContainerSupport.EnvVar(), "true")

			deployment := &storage.Deployment{
				Id:         "deploy-1",
				Name:       "test-deploy",
				Namespace:  "default",
				ClusterId:  "cluster-1",
				Containers: tc.containers,
			}
			process := &storage.ProcessIndicator{
				ContainerName: tc.processContainer,
				Signal:        &storage.ProcessSignal{ExecFilePath: "/bin/sh"},
			}

			obj, err := ConstructDeploymentWithProcess(deployment, tc.images, &NetworkPoliciesApplied{}, process, false)
			require.NoError(t, err)
			require.NotNil(t, obj)
		})
	}
}
