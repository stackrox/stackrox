package aggregator

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
)

var (
	allDeployments = []string{"dep_0", "dep_1", "dep_2", "dep_3"}
	allImages      = []string{"image_0", "image_1", "image_2"}
	indicators1    = []*storage.ProcessIndicator{
		{
			Id:            uuid.NewV4().String(),
			DeploymentId:  allDeployments[0],
			ContainerName: "container_a",
			ImageId:       allImages[0],
			Signal: &storage.ProcessSignal{
				ContainerId:  "13ea7ce738f4",
				Pid:          15,
				Name:         "ssh",
				ExecFilePath: "/usr/bin/ssh",
			},
		},
		{
			Id:            uuid.NewV4().String(),
			DeploymentId:  allDeployments[1],
			ContainerName: "container_b",
			ImageId:       allImages[0],
			Signal: &storage.ProcessSignal{
				ContainerId:  "860a6347711e",
				Pid:          32,
				Name:         "sshd",
				ExecFilePath: "/bin/sshd",
			},
		},
		{
			Id:            uuid.NewV4().String(),
			DeploymentId:  allDeployments[2],
			ContainerName: "container_c",
			ImageId:       allImages[1],
			Signal: &storage.ProcessSignal{
				ContainerId:  "828b7beae96b",
				Pid:          16,
				Name:         "ssh",
				ExecFilePath: "/bin/bash",
			},
		},
		{
			Id:            uuid.NewV4().String(),
			DeploymentId:  allDeployments[3],
			ContainerName: "container_d",
			ImageId:       allImages[1],
			Signal: &storage.ProcessSignal{
				ContainerId:  "17e5fdec203e",
				Pid:          33,
				Name:         "sshd",
				ExecFilePath: "/bin/zsh",
			},
		},
	}

	indicators2 = []*storage.ProcessIndicator{
		{
			Id:            uuid.NewV4().String(),
			DeploymentId:  allDeployments[2],
			ContainerName: "container_single",
			ImageId:       allImages[1],
			Signal: &storage.ProcessSignal{
				ContainerId:  "828b7beae69b",
				Pid:          17,
				Name:         "sh",
				ExecFilePath: "/bin/sh",
			},
		},
		{
			Id:            uuid.NewV4().String(),
			DeploymentId:  allDeployments[2],
			ContainerName: "container_c",
			ImageId:       allImages[1],
			Signal: &storage.ProcessSignal{
				ContainerId:  "828b7beae96b",
				Pid:          16,
				Name:         "ssh",
				ExecFilePath: "/bin/bash",
			},
		},
		{
			Id:            uuid.NewV4().String(),
			DeploymentId:  allDeployments[2],
			ContainerName: "container_c",
			ImageId:       allImages[1],
			Signal: &storage.ProcessSignal{
				ContainerId:  "828b7beae96b",
				Pid:          33,
				Name:         "zsh",
				ExecFilePath: "/bin/zsh",
			},
		},
	}
)

func TestAggregator(t *testing.T) {
	t.Setenv(features.ActiveVulnMgmt.EnvVar(), "true")

	aggregator := &aggregatorImpl{cache: make(map[string]map[string]*ProcessUpdate)}
	mockImageCache := set.NewStringSet()
	scannedImageFunc := func(imageID string) bool {
		return mockImageCache.Contains(imageID)
	}

	// Test case 1: when deployment is removed, cache for deployment is removed.
	aggregator.Add(indicators1[:2])
	assert.Len(t, aggregator.cache, 2)
	deployToupdates := aggregator.GetAndPrune(scannedImageFunc, set.NewStringSet())
	assert.Len(t, aggregator.cache, 0)
	assert.Len(t, deployToupdates, 0)

	// Test case 2: when image is ready but deployment is removed, cache for deployment is removed.
	aggregator.Add(indicators1[:2])
	assert.Len(t, aggregator.cache, 2)
	mockImageCache = set.NewStringSet(indicators1[0].GetImageId(), indicators1[1].GetImageId())
	deployToupdates = aggregator.GetAndPrune(scannedImageFunc, set.NewStringSet())
	assert.Len(t, aggregator.cache, 0)
	assert.Len(t, deployToupdates, 0)

	// Test case 3: when image is not ready, no update to process.
	aggregator.Add(indicators1[:2])
	assert.Len(t, aggregator.cache, 2)
	mockImageCache = set.NewStringSet()
	deployToupdates = aggregator.GetAndPrune(scannedImageFunc, set.NewStringSet(indicators1[0].GetDeploymentId(), indicators1[1].GetDeploymentId()))
	assert.Len(t, deployToupdates, 0)
	assert.Len(t, aggregator.cache, 2)

	// Test case 4: when image is ready, generate update to fetch from Database.
	mockImageCache = set.NewStringSet(indicators1[0].GetImageId(), indicators1[1].GetImageId())
	deployToupdates = aggregator.GetAndPrune(scannedImageFunc, set.NewStringSet(indicators1[0].GetDeploymentId(), indicators1[1].GetDeploymentId()))
	assert.Len(t, deployToupdates, 2)
	for _, updates := range deployToupdates {
		assert.Len(t, updates, 1)
		assert.True(t, updates[0].FromDatabase())
	}
	assert.Len(t, aggregator.cache, 2)
	aggregator.Add(indicators1)

	mockImageCache = set.NewStringSet(allImages...)
	existingDeployments := set.NewStringSet(allDeployments...)
	deployToupdates = aggregator.GetAndPrune(scannedImageFunc, existingDeployments)
	assert.Len(t, aggregator.cache, 4)
	for _, containerMap := range aggregator.cache {
		for _, update := range containerMap {
			assert.Equal(t, update.state, FromCache)
		}
	}
	assert.Len(t, deployToupdates, 4)
	for idx, indicator := range indicators1 {
		assert.Contains(t, deployToupdates, indicator.GetDeploymentId())
		assert.Len(t, deployToupdates[indicator.GetDeploymentId()], 1)
		assert.Equal(t, indicator.GetContainerName(), deployToupdates[indicator.GetDeploymentId()][0].ContainerName)
		if idx < 2 {
			assert.True(t, deployToupdates[indicator.GetDeploymentId()][0].FromCache())
		} else {
			assert.True(t, deployToupdates[indicator.GetDeploymentId()][0].FromDatabase())
		}
	}

	// Test case 5: No new indicators, no update
	deployToupdates = aggregator.GetAndPrune(scannedImageFunc, existingDeployments)
	assert.Len(t, deployToupdates, 0)
	for _, updates := range deployToupdates {
		assert.True(t, updates[0].FromDatabase())
	}

	// Test case 6: New indicators coming, keep generating updates
	aggregator.Add(indicators2)
	deployToupdates = aggregator.GetAndPrune(scannedImageFunc, existingDeployments)
	assert.Len(t, deployToupdates, 1)
	assert.Contains(t, deployToupdates, indicators2[0].GetDeploymentId())
	updates := deployToupdates[indicators2[0].GetDeploymentId()]
	assert.Len(t, updates, 2)

	for _, update := range updates {
		if update.ContainerName == indicators2[0].GetContainerName() {
			assert.Equal(t, 0, update.NewPaths.Cardinality())
			assert.True(t, update.FromDatabase())
		} else {
			assert.Equal(t, update.ContainerName, indicators2[1].GetContainerName())
			assert.Equal(t, 2, update.NewPaths.Cardinality())
			assert.Contains(t, update.NewPaths, indicators2[1].GetSignal().GetExecFilePath())
			assert.Contains(t, update.NewPaths, indicators2[2].GetSignal().GetExecFilePath())
		}
	}

	// Test case 7: Container removed from deployment, generate delete update.
	newDeployment := &storage.Deployment{
		Id: allDeployments[2],
		Containers: []*storage.Container{
			{
				Name:  "container_c",
				Image: &storage.ContainerImage{Id: allImages[1]},
			},
		},
	}
	containerToRemove := "container_single"
	aggregator.RefreshDeployment(newDeployment)

	deployToupdates = aggregator.GetAndPrune(scannedImageFunc, existingDeployments)
	assert.Len(t, deployToupdates, 1)
	updates = deployToupdates[indicators2[0].GetDeploymentId()]
	assert.Len(t, updates, 1)
	assert.Equal(t, containerToRemove, updates[0].ContainerName)
	assert.True(t, updates[0].ToBeRemoved())
	assert.NotContains(t, aggregator.cache[indicators2[0].GetDeploymentId()], containerToRemove)

	// Test case 8: Indicators coming in with wrong image, no update
	indicatorNewImage := indicators2[2].Clone()
	indicatorNewImage.ImageId = allImages[0]
	aggregator.Add([]*storage.ProcessIndicator{indicatorNewImage})
	deployToupdates = aggregator.GetAndPrune(scannedImageFunc, existingDeployments)
	assert.Len(t, deployToupdates, 0)

	// Test case 9: Container image changed, generate update from database
	newDeployment = &storage.Deployment{
		Id: allDeployments[2],
		Containers: []*storage.Container{
			{
				Name:  "container_c",
				Image: &storage.ContainerImage{Id: allImages[0]},
			},
		},
	}
	aggregator.RefreshDeployment(newDeployment)
	deployToupdates = aggregator.GetAndPrune(scannedImageFunc, existingDeployments)
	updates = deployToupdates[indicators2[0].GetDeploymentId()]
	assert.Len(t, updates, 1)
	assert.True(t, updates[0].FromDatabase())
}
