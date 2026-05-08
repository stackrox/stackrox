package augmentedobjs

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterInitContainers(t *testing.T) {
	initContainer := &storage.Container{
		Name: "init",
		Type: storage.ContainerType_INIT,
	}
	regularContainer := &storage.Container{
		Name: "main",
		Type: storage.ContainerType_REGULAR,
	}
	initImage := &storage.Image{
		Id: "init-image",
	}
	regularImage := &storage.Image{
		Id: "main-image",
	}

	cases := map[string]struct {
		flagEnabled        bool
		containers         []*storage.Container
		images             []*storage.Image
		expectedContainers []*storage.Container
		expectedImages     []*storage.Image
	}{
		"flag enabled, init containers filtered out": {
			flagEnabled:        true,
			containers:         []*storage.Container{initContainer, regularContainer},
			images:             []*storage.Image{initImage, regularImage},
			expectedContainers: []*storage.Container{regularContainer},
			expectedImages:     []*storage.Image{regularImage},
		},
		"flag enabled, no init containers is a no-op": {
			flagEnabled:        true,
			containers:         []*storage.Container{regularContainer},
			images:             []*storage.Image{regularImage},
			expectedContainers: []*storage.Container{regularContainer},
			expectedImages:     []*storage.Image{regularImage},
		},
		"flag disabled, init containers passed through": {
			flagEnabled:        false,
			containers:         []*storage.Container{initContainer, regularContainer},
			images:             []*storage.Image{initImage, regularImage},
			expectedContainers: []*storage.Container{initContainer, regularContainer},
			expectedImages:     []*storage.Image{initImage, regularImage},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if tc.flagEnabled {
				t.Setenv(features.InitContainerSupport.EnvVar(), "true")
			} else {
				t.Setenv(features.InitContainerSupport.EnvVar(), "false")
			}

			deployment := &storage.Deployment{
				Containers: tc.containers,
			}

			filtered, filteredImages := filterInitContainers(deployment, tc.images)

			require.Len(t, filtered.GetContainers(), len(tc.expectedContainers))
			for i, c := range filtered.GetContainers() {
				assert.Equal(t, tc.expectedContainers[i].GetName(), c.GetName())
				assert.Equal(t, tc.expectedContainers[i].GetType(), c.GetType())
			}
			require.Len(t, filteredImages, len(tc.expectedImages))
			for i, img := range filteredImages {
				assert.Equal(t, tc.expectedImages[i].GetId(), img.GetId())
			}
		})
	}
}

func TestConstructDeploymentFiltersInitContainers(t *testing.T) {
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

func TestConstructDeploymentWithProcessFiltersInitContainers(t *testing.T) {
	cases := map[string]struct {
		containers       []*storage.Container
		images           []*storage.Image
		processContainer string
		expectError      bool
	}{
		"single init + single regular, process on regular": {
			containers: []*storage.Container{
				{Name: "init", Type: storage.ContainerType_INIT},
				{Name: "main", Type: storage.ContainerType_REGULAR},
			},
			images:           []*storage.Image{{Id: "init-img"}, {Id: "main-img"}},
			processContainer: "main",
		},
		"init between two regular containers, process on second regular": {
			containers: []*storage.Container{
				{Name: "app", Type: storage.ContainerType_REGULAR},
				{Name: "init-setup", Type: storage.ContainerType_INIT},
				{Name: "sidecar", Type: storage.ContainerType_REGULAR},
			},
			images:           []*storage.Image{{Id: "app-img"}, {Id: "init-img"}, {Id: "sidecar-img"}},
			processContainer: "sidecar",
		},
		"init between two regular containers, process on first regular": {
			containers: []*storage.Container{
				{Name: "app", Type: storage.ContainerType_REGULAR},
				{Name: "init-setup", Type: storage.ContainerType_INIT},
				{Name: "sidecar", Type: storage.ContainerType_REGULAR},
			},
			images:           []*storage.Image{{Id: "app-img"}, {Id: "init-img"}, {Id: "sidecar-img"}},
			processContainer: "app",
		},
		"multiple init containers before multiple regular containers": {
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
		"multiple init containers + single regular, process on regular": {
			containers: []*storage.Container{
				{Name: "init-db", Type: storage.ContainerType_INIT},
				{Name: "init-config", Type: storage.ContainerType_INIT},
				{Name: "main", Type: storage.ContainerType_REGULAR},
			},
			images:           []*storage.Image{{Id: "init-db-img"}, {Id: "init-config-img"}, {Id: "main-img"}},
			processContainer: "main",
		},
		"process on filtered init container returns error": {
			containers: []*storage.Container{
				{Name: "init", Type: storage.ContainerType_INIT},
				{Name: "main", Type: storage.ContainerType_REGULAR},
			},
			images:           []*storage.Image{{Id: "init-img"}, {Id: "main-img"}},
			processContainer: "init",
			expectError:      true,
		},
		"no init containers, multiple regular, process on last": {
			containers: []*storage.Container{
				{Name: "app", Type: storage.ContainerType_REGULAR},
				{Name: "sidecar", Type: storage.ContainerType_REGULAR},
				{Name: "proxy", Type: storage.ContainerType_REGULAR},
			},
			images:           []*storage.Image{{Id: "app-img"}, {Id: "sidecar-img"}, {Id: "proxy-img"}},
			processContainer: "proxy",
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
			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not found")
			} else {
				require.NoError(t, err)
				require.NotNil(t, obj)
			}
		})
	}
}

func TestConstructDeploymentWithProcessFlagOff(t *testing.T) {
	t.Setenv(features.InitContainerSupport.EnvVar(), "false")

	deployment := &storage.Deployment{
		Id:        "deploy-1",
		Name:      "test-deploy",
		Namespace: "default",
		ClusterId: "cluster-1",
		Containers: []*storage.Container{
			{Name: "init-db", Type: storage.ContainerType_INIT},
			{Name: "init-config", Type: storage.ContainerType_INIT},
			{Name: "api", Type: storage.ContainerType_REGULAR},
			{Name: "sidecar", Type: storage.ContainerType_REGULAR},
		},
	}
	images := []*storage.Image{
		{Id: "init-db-img"},
		{Id: "init-config-img"},
		{Id: "api-img"},
		{Id: "sidecar-img"},
	}

	// With flag off, init containers are NOT filtered. Process on a regular container
	// should still resolve correctly against the full unfiltered list.
	process := &storage.ProcessIndicator{
		ContainerName: "sidecar",
		Signal:        &storage.ProcessSignal{ExecFilePath: "/bin/sh"},
	}
	obj, err := ConstructDeploymentWithProcess(deployment, images, &NetworkPoliciesApplied{}, process, false)
	require.NoError(t, err)
	require.NotNil(t, obj)

	// Process on an init container should also resolve when flag is off (no filtering).
	processOnInit := &storage.ProcessIndicator{
		ContainerName: "init-db",
		Signal:        &storage.ProcessSignal{ExecFilePath: "/bin/sh"},
	}
	obj, err = ConstructDeploymentWithProcess(deployment, images, &NetworkPoliciesApplied{}, processOnInit, false)
	require.NoError(t, err)
	require.NotNil(t, obj)
}
