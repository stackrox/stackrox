package updater

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	acConverter "github.com/stackrox/rox/central/activecomponent/converter"
	acMocks "github.com/stackrox/rox/central/activecomponent/datastore/mocks"
	aggregatorPkg "github.com/stackrox/rox/central/activecomponent/updater/aggregator"
	"github.com/stackrox/rox/central/activecomponent/updater/aggregator/mocks"
	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	imageMocks "github.com/stackrox/rox/central/image/datastore/mocks"
	piMocks "github.com/stackrox/rox/central/processindicator/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/simplecache"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type indicatorModel struct {
	DeploymentID  string
	ContainerName string
	ImageID       string
	ExePaths      []string
}

var (
	mockDeployments = []*storage.Deployment{
		{
			Id: "depA",
			Containers: []*storage.Container{
				{
					Name:  "depA-C1-image1",
					Image: &storage.ContainerImage{Id: "image1"},
				},
				{
					Name:  "depA-C2-image1",
					Image: &storage.ContainerImage{Id: "image1"},
				},
			},
		},
		{
			Id: "depB",
			Containers: []*storage.Container{
				{
					Name:  "depB-C1-image2",
					Image: &storage.ContainerImage{Id: "image2"},
				},
				{
					Name:  "depB-C2-image1",
					Image: &storage.ContainerImage{Id: "image1"},
				},
			},
		},
		{
			Id: "depC",
			Containers: []*storage.Container{
				{
					Name:  "depC-C1-image2",
					Image: &storage.ContainerImage{Id: "image2"},
				},
			},
		},
	}
	mockImage = &storage.Image{
		Id: "image1",
		Scan: &storage.ImageScan{
			ScanTime: protoconv.ConvertTimeToTimestamp(time.Now()),
			// leaving empty initially so the test will cover backwards compatibility for scans with no version
			ScannerVersion: "",
			Components: []*storage.EmbeddedImageScanComponent{
				{
					Name:    "image1_component1",
					Version: "1",
					Source:  storage.SourceType_OS,
					Executables: []*storage.EmbeddedImageScanComponent_Executable{
						{Path: "/root/bin/image1_component1_match_file1", Dependencies: []string{scancomponent.ComponentID("image1_component1", "1", "")}},
						{Path: "/root/bin/image1_component1_nonmatch_file2", Dependencies: []string{scancomponent.ComponentID("image1_component1", "1", "")}},
						{Path: "/root/bin/image1_component1_nonmatch_file3", Dependencies: []string{scancomponent.ComponentID("image1_component1", "1", "")}},
					},
				},
				{
					Name:    "image1_component2",
					Version: "2",
					Source:  storage.SourceType_OS,
					Executables: []*storage.EmbeddedImageScanComponent_Executable{
						{Path: "/root/bin/image1_component2_nonmatch_file1", Dependencies: []string{scancomponent.ComponentID("image1_component2", "2", "")}},
						{Path: "/root/bin/image1_component2_nonmatch_file2", Dependencies: []string{scancomponent.ComponentID("image1_component2", "2", "")}},
						{Path: "/root/bin/image1_component2_match_file3", Dependencies: []string{scancomponent.ComponentID("image1_component2", "2", "")}},
					},
				},
				{
					Name:    "image1_component3",
					Version: "2",
					Source:  storage.SourceType_JAVA,
				},
				{
					Name:    "image1_component4",
					Version: "2",
					Source:  storage.SourceType_OS,
					Executables: []*storage.EmbeddedImageScanComponent_Executable{
						{Path: "/root/bin/image1_component4_nonmatch_file1", Dependencies: []string{scancomponent.ComponentID("image1_component4", "2", "")}},
						{Path: "/root/bin/image1_component4_nonmatch_file2", Dependencies: []string{scancomponent.ComponentID("image1_component4", "2", "")}},
						{Path: "/root/bin/image1_component4_match_file3", Dependencies: []string{scancomponent.ComponentID("image1_component4", "2", "")}},
					},
				},
			},
		},
	}

	mockIndicators = []indicatorModel{
		{
			DeploymentID:  "depA",
			ContainerName: "depA-C1-image1",
			ImageID:       mockImage.Id,
			ExePaths: []string{
				"/root/bin/image1_component1_match_file1",
				"/root/bin/image1_component2_match_file3",
				"/root/bin/image1_component3_match_file1",
				"/root/bin/image1_component3_match_file2",
			},
		},
		{
			DeploymentID:  "depB",
			ContainerName: "depB-C2-image1",
			ImageID:       mockImage.Id,
			ExePaths: []string{
				"/root/bin/image1_component1_match_file1",
				"/root/bin/image1_component3_match_file3",
				"/root/bin/image1_component4_match_file3",
			},
		},
	}
)

func TestActiveComponentUpdater(t *testing.T) {
	t.Setenv(features.ActiveVulnMgmt.EnvVar(), "true")
	suite.Run(t, new(acUpdaterTestSuite))
}

type acUpdaterTestSuite struct {
	suite.Suite

	mockCtrl                      *gomock.Controller
	mockDeploymentDatastore       *deploymentMocks.MockDataStore
	mockActiveComponentDataStore  *acMocks.MockDataStore
	mockProcessIndicatorDataStore *piMocks.MockDataStore
	mockImageDataStore            *imageMocks.MockDataStore
	executableCache               simplecache.Cache
	mockAggregator                *mocks.MockProcessAggregator
}

func (s *acUpdaterTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockDeploymentDatastore = deploymentMocks.NewMockDataStore(s.mockCtrl)
	s.mockActiveComponentDataStore = acMocks.NewMockDataStore(s.mockCtrl)
	s.mockProcessIndicatorDataStore = piMocks.NewMockDataStore(s.mockCtrl)
	s.mockImageDataStore = imageMocks.NewMockDataStore(s.mockCtrl)
	s.executableCache = simplecache.New()
	s.mockAggregator = mocks.NewMockProcessAggregator(s.mockCtrl)
}

func (s *acUpdaterTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *acUpdaterTestSuite) assertHasContainer(contexts []*storage.ActiveComponent_ActiveContext, container string) {
	var found bool
	for _, ctx := range contexts {
		if ctx.GetContainerName() == container {
			found = true
			break
		}
	}
	s.True(found)
}

func (s *acUpdaterTestSuite) TestUpdater() {
	imageID := "image1"
	updater := &updaterImpl{
		acStore:         s.mockActiveComponentDataStore,
		deploymentStore: s.mockDeploymentDatastore,
		piStore:         s.mockProcessIndicatorDataStore,
		imageStore:      s.mockImageDataStore,
		aggregator:      aggregatorPkg.NewAggregator(),
		executableCache: simplecache.New(),
	}
	var deploymentIDs []string
	for _, deployment := range mockDeployments {
		for _, container := range deployment.GetContainers() {
			if container.GetImage().GetId() == imageID {
				deploymentIDs = append(deploymentIDs, deployment.GetId())
			}
		}
	}
	s.mockDeploymentDatastore.EXPECT().GetDeploymentIDs(gomock.Any()).AnyTimes().Return(deploymentIDs, nil)
	s.mockActiveComponentDataStore.EXPECT().SearchRawActiveComponents(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
	s.mockProcessIndicatorDataStore.EXPECT().SearchRawProcessIndicators(gomock.Any(), gomock.Any()).Times(3).DoAndReturn(
		func(ctx context.Context, query *v1.Query) ([]*storage.ProcessIndicator, error) {
			queries := query.GetConjunction().GetQueries()
			s.Assert().Len(queries, 2)
			var containerName, deploymentID string
			for _, q := range queries {
				mf := q.GetBaseQuery().GetMatchFieldQuery()

				switch mf.GetField() {
				case search.DeploymentID.String():
					deploymentID = stripQuotes(mf.GetValue())
				case search.ContainerName.String():
					containerName = stripQuotes(mf.GetValue())
				default:
					s.Assert().Fail("unexpected query")
				}
			}
			for _, pi := range mockIndicators {
				if pi.ContainerName == containerName && deploymentID == pi.DeploymentID {
					var ret []*storage.ProcessIndicator
					for _, exec := range pi.ExePaths {
						ret = append(ret, &storage.ProcessIndicator{
							Id:      uuid.NewV4().String(),
							ImageId: pi.ImageID,
							Signal:  &storage.ProcessSignal{ExecFilePath: exec}},
						)
					}
					return ret, nil
				}
			}
			return nil, nil
		})
	s.mockImageDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(ctx context.Context, query *v1.Query) ([]search.Result, error) {
			return []search.Result{{ID: imageID}}, nil
		})
	s.mockActiveComponentDataStore.EXPECT().UpsertBatch(gomock.Any(), gomock.Any()).AnyTimes().Return(nil).Do(func(_ context.Context, acs []*storage.ActiveComponent) {
		s.Assert().Equal(2, len(acs))
		for _, ac := range acs {
			// Deployment C does not have image1.
			s.Assert().NotEqual(ac.GetDeploymentId(), mockDeployments[2].GetId())

			imageComponent := pgSearch.IDToParts(ac.GetComponentId())[0]

			s.Assert().True(strings.HasPrefix(imageComponent, mockImage.GetId()))
			s.Assert().NotEqual(imageComponent, mockImage.GetScan().GetComponents()[2].GetName())
			s.Assert().Len(ac.GetActiveContextsSlice(), 1)

			var expectedComponent *storage.EmbeddedImageScanComponent
			var expectedContainer string
			if ac.GetDeploymentId() == mockDeployments[0].Id {
				expectedContainer = mockIndicators[0].ContainerName
				// Component 1 or 2
				expectedComponent = mockImage.GetScan().GetComponents()[0]
				if imageComponent != mockImage.GetScan().GetComponents()[0].GetName() {
					expectedComponent = mockImage.GetScan().GetComponents()[1]
				}
			} else {
				s.Assert().Equal(ac.GetDeploymentId(), mockDeployments[1].Id)
				expectedContainer = mockIndicators[1].ContainerName
				// Component 1 or 4
				expectedComponent = mockImage.GetScan().GetComponents()[0]
				if imageComponent != mockImage.GetScan().GetComponents()[0].GetName() {
					expectedComponent = mockImage.GetScan().GetComponents()[3]
				}
			}
			s.assertHasContainer(ac.GetActiveContextsSlice(), expectedContainer)
			s.Assert().True(strings.HasSuffix(imageComponent, expectedComponent.GetName()))
			s.Assert().Equal(ac.GetComponentId(), scancomponent.ComponentID(expectedComponent.GetName(), expectedComponent.GetVersion(), ""))
		}
	})

	s.Assert().NoError(updater.PopulateExecutableCache(updaterCtx, mockImage))
	for _, deployment := range mockDeployments {
		updater.aggregator.RefreshDeployment(deployment)
	}
	updater.Update()
}

func (s *acUpdaterTestSuite) TestUpdater_PopulateExecutableCache() {
	updater := &updaterImpl{
		acStore:         s.mockActiveComponentDataStore,
		deploymentStore: s.mockDeploymentDatastore,
		piStore:         s.mockProcessIndicatorDataStore,
		imageStore:      s.mockImageDataStore,
		aggregator:      s.mockAggregator,
		executableCache: simplecache.New(),
	}

	// Initial population
	// scanner version is empty to test backward compatibility
	image := mockImage.Clone()
	s.Assert().Equal(image.GetScan().GetScannerVersion(), "")
	s.Assert().NoError(updater.PopulateExecutableCache(updaterCtx, image))
	s.verifyExecutableCache(updater, mockImage)

	// Verify the executables are not stored.
	for _, component := range image.GetScan().GetComponents() {
		s.Assert().Empty(component.Executables)
	}

	// Image won't be processed again.
	s.Assert().NoError(updater.PopulateExecutableCache(updaterCtx, image))
	s.verifyExecutableCache(updater, mockImage)

	// New update without the first component
	image = mockImage.Clone()
	// update the scanner version to make sure cache gets re-populated
	image.GetScan().ScannerVersion = "2.22.0"
	image.GetScan().Components = image.GetScan().GetComponents()[1:]
	imageForVerify := image.Clone()
	s.Assert().NoError(updater.PopulateExecutableCache(updaterCtx, image))
	s.verifyExecutableCache(updater, imageForVerify)
}

func (s *acUpdaterTestSuite) verifyExecutableCache(updater *updaterImpl, image *storage.Image) {
	s.Assert().Len(updater.executableCache.Keys(), 1)

	result, ok := updater.executableCache.Get(image.GetId())
	s.Assert().True(ok)

	// ensure the scanner version is updated and matches
	s.Assert().Equal(image.GetScan().GetScannerVersion(), result.(*imageExecutable).scannerVersion)
	execToComponents := result.(*imageExecutable).execToComponents
	allExecutables := set.NewStringSet()
	for _, component := range image.GetScan().GetComponents() {
		if component.Source != storage.SourceType_OS {
			continue
		}
		componentID := scancomponent.ComponentID(component.GetName(), component.GetVersion(), "")
		for _, exec := range component.Executables {
			s.Assert().Contains(execToComponents, exec.GetPath())
			s.Assert().Len(execToComponents[exec.GetPath()], 1)
			s.Assert().Equal(componentID, execToComponents[exec.GetPath()][0])
			allExecutables.Add(exec.GetPath())
		}
	}
	s.Assert().Len(execToComponents, len(allExecutables))
}
func (s *acUpdaterTestSuite) TestUpdater_Update() {
	updater := &updaterImpl{
		acStore:         s.mockActiveComponentDataStore,
		deploymentStore: s.mockDeploymentDatastore,
		piStore:         s.mockProcessIndicatorDataStore,
		imageStore:      s.mockImageDataStore,
		aggregator:      s.mockAggregator,
		executableCache: simplecache.New(),
	}
	image := &storage.Image{
		Id: "image1",
		Scan: &storage.ImageScan{
			ScanTime:       protoconv.ConvertTimeToTimestamp(time.Now()),
			ScannerVersion: "2.22.0",
			Components: []*storage.EmbeddedImageScanComponent{
				{
					Name:    "component1",
					Version: "1",
					Source:  storage.SourceType_OS,
					Executables: []*storage.EmbeddedImageScanComponent_Executable{
						{Path: "/usr/bin/component1_file1", Dependencies: []string{scancomponent.ComponentID("component1", "1", "")}},
						{Path: "/usr/bin/component1_file2", Dependencies: []string{scancomponent.ComponentID("component1", "1", "")}},
						{Path: "/usr/bin/component1and2_file3", Dependencies: []string{scancomponent.ComponentID("component1", "1", "")}},
						{Path: "/usr/bin/component1_file4", Dependencies: []string{
							scancomponent.ComponentID("component1", "1", ""),
							scancomponent.ComponentID("component2", "1", ""),
						}},
					},
				},
				{
					Name:    "component2",
					Version: "1",
					Source:  storage.SourceType_OS,
					Executables: []*storage.EmbeddedImageScanComponent_Executable{
						{Path: "/usr/bin/component2_file1", Dependencies: []string{scancomponent.ComponentID("component2", "1", "")}},
						{Path: "/usr/bin/component2_file2", Dependencies: []string{scancomponent.ComponentID("component2", "1", "")}},
						{Path: "/usr/bin/component1and2_file3", Dependencies: []string{scancomponent.ComponentID("component2", "1", "")}},
					},
				},
			},
		},
	}
	imageScan := image.GetScan()
	components := imageScan.GetComponents()
	deployment := mockDeployments[0]
	var componentsIDs []string
	for _, component := range components {
		componentsIDs = append(componentsIDs, scancomponent.ComponentID(component.GetName(), component.GetVersion(), ""))
	}

	var containerNames []string
	for _, container := range deployment.GetContainers() {
		containerNames = append(containerNames, container.GetName())
	}

	s.mockImageDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx context.Context, query *v1.Query) ([]search.Result, error) {
			return []search.Result{{ID: image.GetId()}}, nil
		})
	s.Assert().NoError(updater.PopulateExecutableCache(updaterCtx, image.Clone()))
	s.mockDeploymentDatastore.EXPECT().GetDeploymentIDs(gomock.Any()).AnyTimes().Return([]string{deployment.GetId()}, nil)

	// Test active components with designated image and deployment
	var testCases = []struct {
		description string

		updates     []*aggregatorPkg.ProcessUpdate
		indicators  map[string]indicatorModel
		existingAcs map[string]set.StringSet // componentID to container name map

		acsToUpdate map[string]set.StringSet // expected Acs to be updated, componentID to container name map
		acsToDelete []string
		imageChange bool
	}{
		{
			description: "First populate from database",
			updates: []*aggregatorPkg.ProcessUpdate{
				aggregatorPkg.NewProcessUpdate(image.GetId(), containerNames[0], set.NewStringSet(), aggregatorPkg.FromDatabase),
				aggregatorPkg.NewProcessUpdate(image.GetId(), containerNames[1], set.NewStringSet(), aggregatorPkg.FromDatabase),
			},
			indicators: map[string]indicatorModel{
				containerNames[0]: {
					DeploymentID:  deployment.GetId(),
					ContainerName: containerNames[0],
					ImageID:       image.GetId(),
					ExePaths: []string{
						components[0].Executables[0].Path,
						components[1].Executables[1].Path,
					},
				},
				containerNames[1]: {
					DeploymentID:  deployment.GetId(),
					ContainerName: containerNames[1],
					ImageID:       image.GetId(),
					ExePaths: []string{
						components[1].Executables[0].Path,
					},
				},
			},
			acsToUpdate: map[string]set.StringSet{
				componentsIDs[0]: set.NewStringSet(deployment.Containers[0].GetName()),
				componentsIDs[1]: set.NewStringSet(deployment.Containers[0].GetName(), deployment.Containers[1].GetName()),
			},
		},
		{
			description: "Restart and populate from database no updates",
			updates: []*aggregatorPkg.ProcessUpdate{
				aggregatorPkg.NewProcessUpdate(image.GetId(), containerNames[0], set.NewStringSet(), aggregatorPkg.FromDatabase),
				aggregatorPkg.NewProcessUpdate(image.GetId(), containerNames[1], set.NewStringSet(), aggregatorPkg.FromDatabase),
			},
			indicators: map[string]indicatorModel{
				containerNames[0]: {
					DeploymentID:  deployment.GetId(),
					ContainerName: containerNames[0],
					ImageID:       image.GetId(),
					ExePaths: []string{
						components[0].Executables[0].Path,
						components[1].Executables[1].Path,
					},
				},
				containerNames[1]: {
					DeploymentID:  deployment.GetId(),
					ContainerName: containerNames[1],
					ImageID:       image.GetId(),
					ExePaths: []string{
						components[1].Executables[0].Path,
					},
				},
			},
			existingAcs: map[string]set.StringSet{
				componentsIDs[0]: set.NewStringSet(deployment.Containers[0].GetName()),
				componentsIDs[1]: set.NewStringSet(deployment.Containers[0].GetName(), deployment.Containers[1].GetName()),
			},
		},
		{
			description: "Restart and populate from database with updates",
			updates: []*aggregatorPkg.ProcessUpdate{
				aggregatorPkg.NewProcessUpdate(image.GetId(), containerNames[0], set.NewStringSet(), aggregatorPkg.FromDatabase),
				aggregatorPkg.NewProcessUpdate(image.GetId(), containerNames[1], set.NewStringSet(), aggregatorPkg.FromDatabase),
			},
			indicators: map[string]indicatorModel{
				containerNames[0]: {
					DeploymentID:  deployment.GetId(),
					ContainerName: containerNames[0],
					ImageID:       image.GetId(),
					ExePaths: []string{
						components[0].Executables[0].Path,
						components[1].Executables[1].Path,
					},
				},
				containerNames[1]: {
					DeploymentID:  deployment.GetId(),
					ContainerName: containerNames[1],
					ImageID:       image.GetId(),
					ExePaths: []string{
						components[1].Executables[0].Path,
					},
				},
			},
			existingAcs: map[string]set.StringSet{
				componentsIDs[0]: set.NewStringSet(deployment.Containers[1].GetName()),
				componentsIDs[1]: set.NewStringSet(deployment.Containers[0].GetName()),
			},
			acsToUpdate: map[string]set.StringSet{
				componentsIDs[0]: set.NewStringSet(deployment.Containers[0].GetName()),
				componentsIDs[1]: set.NewStringSet(deployment.Containers[0].GetName(), deployment.Containers[1].GetName()),
			},
		},
		{
			description: "Restart and populate from database with removal",
			updates: []*aggregatorPkg.ProcessUpdate{
				aggregatorPkg.NewProcessUpdate(image.GetId(), containerNames[0], set.NewStringSet(), aggregatorPkg.FromDatabase),
				aggregatorPkg.NewProcessUpdate(image.GetId(), containerNames[1], set.NewStringSet(), aggregatorPkg.FromDatabase),
			},
			indicators: map[string]indicatorModel{
				containerNames[0]: {
					DeploymentID:  deployment.GetId(),
					ContainerName: containerNames[0],
					ImageID:       image.GetId(),
					ExePaths: []string{
						components[1].Executables[0].Path,
						components[1].Executables[1].Path,
					},
				},
				containerNames[1]: {
					DeploymentID:  deployment.GetId(),
					ContainerName: containerNames[1],
					ImageID:       image.GetId(),
					ExePaths: []string{
						components[1].Executables[0].Path,
					},
				},
			},
			existingAcs: map[string]set.StringSet{
				componentsIDs[0]: set.NewStringSet(deployment.Containers[1].GetName()),
				componentsIDs[1]: set.NewStringSet(deployment.Containers[0].GetName()),
			},
			acsToUpdate: map[string]set.StringSet{
				componentsIDs[1]: set.NewStringSet(deployment.Containers[0].GetName(), deployment.Containers[1].GetName()),
			},
			acsToDelete: []string{componentsIDs[0]},
		},
		{
			description: "Restart and populate from database with removal request",
			updates: []*aggregatorPkg.ProcessUpdate{
				aggregatorPkg.NewProcessUpdate(image.GetId(), containerNames[0], set.NewStringSet(), aggregatorPkg.ToBeRemoved),
				aggregatorPkg.NewProcessUpdate(image.GetId(), containerNames[1], set.NewStringSet(), aggregatorPkg.FromDatabase),
			},
			indicators: map[string]indicatorModel{
				containerNames[0]: {
					DeploymentID:  deployment.GetId(),
					ContainerName: containerNames[0],
					ImageID:       image.GetId(),
					ExePaths: []string{
						components[1].Executables[0].Path,
						components[1].Executables[1].Path,
					},
				},
				containerNames[1]: {
					DeploymentID:  deployment.GetId(),
					ContainerName: containerNames[1],
					ImageID:       image.GetId(),
					ExePaths: []string{
						components[0].Executables[0].Path,
					},
				},
			},
			existingAcs: map[string]set.StringSet{
				componentsIDs[0]: set.NewStringSet(deployment.Containers[1].GetName()),
				componentsIDs[1]: set.NewStringSet(deployment.Containers[0].GetName()),
			},
			acsToDelete: []string{componentsIDs[1]},
		},
		{
			description: "Image change populate from database with updates",
			updates: []*aggregatorPkg.ProcessUpdate{
				aggregatorPkg.NewProcessUpdate(image.GetId(), containerNames[0], set.NewStringSet(), aggregatorPkg.FromDatabase),
				aggregatorPkg.NewProcessUpdate(image.GetId(), containerNames[1], set.NewStringSet(), aggregatorPkg.FromDatabase),
			},
			indicators: map[string]indicatorModel{
				containerNames[0]: {
					DeploymentID:  deployment.GetId(),
					ContainerName: containerNames[0],
					ImageID:       image.GetId(),
					ExePaths: []string{
						components[0].Executables[0].Path,
						components[1].Executables[1].Path,
					},
				},
				containerNames[1]: {
					DeploymentID:  deployment.GetId(),
					ContainerName: containerNames[1],
					ImageID:       image.GetId(),
					ExePaths: []string{
						components[1].Executables[0].Path,
					},
				},
			},
			existingAcs: map[string]set.StringSet{
				componentsIDs[0]: set.NewStringSet(deployment.Containers[0].GetName()),
				componentsIDs[1]: set.NewStringSet(deployment.Containers[0].GetName(), deployment.Containers[1].GetName()),
			},
			acsToUpdate: map[string]set.StringSet{
				componentsIDs[0]: set.NewStringSet(deployment.Containers[0].GetName()),
				componentsIDs[1]: set.NewStringSet(deployment.Containers[0].GetName(), deployment.Containers[1].GetName()),
			},
			imageChange: true,
		},
		{
			description: "Image change populate from database with updates",
			updates: []*aggregatorPkg.ProcessUpdate{
				aggregatorPkg.NewProcessUpdate(image.GetId(), containerNames[0], set.NewStringSet(), aggregatorPkg.FromDatabase),
				aggregatorPkg.NewProcessUpdate(image.GetId(), containerNames[1], set.NewStringSet(), aggregatorPkg.FromDatabase),
			},
			indicators: map[string]indicatorModel{
				containerNames[0]: {
					DeploymentID:  deployment.GetId(),
					ContainerName: containerNames[0],
					ImageID:       image.GetId(),
					ExePaths: []string{
						components[0].Executables[0].Path,
						components[1].Executables[1].Path,
					},
				},
				containerNames[1]: {
					DeploymentID:  deployment.GetId(),
					ContainerName: containerNames[1],
					ImageID:       image.GetId(),
					ExePaths: []string{
						components[1].Executables[0].Path,
					},
				},
			},
			existingAcs: map[string]set.StringSet{
				componentsIDs[0]: set.NewStringSet(deployment.Containers[1].GetName()),
				componentsIDs[1]: set.NewStringSet(deployment.Containers[0].GetName()),
			},
			acsToUpdate: map[string]set.StringSet{
				componentsIDs[0]: set.NewStringSet(deployment.Containers[0].GetName()),
				componentsIDs[1]: set.NewStringSet(deployment.Containers[0].GetName(), deployment.Containers[1].GetName()),
			},
			imageChange: true,
		},
		{
			description: "Image change populate from database with removal",
			updates: []*aggregatorPkg.ProcessUpdate{
				aggregatorPkg.NewProcessUpdate(image.GetId(), containerNames[0], set.NewStringSet(), aggregatorPkg.FromDatabase),
				aggregatorPkg.NewProcessUpdate(image.GetId(), containerNames[1], set.NewStringSet(), aggregatorPkg.FromDatabase),
			},
			indicators: map[string]indicatorModel{
				containerNames[0]: {
					DeploymentID:  deployment.GetId(),
					ContainerName: containerNames[0],
					ImageID:       image.GetId(),
					ExePaths: []string{
						components[1].Executables[0].Path,
						components[1].Executables[1].Path,
					},
				},
				containerNames[1]: {
					DeploymentID:  deployment.GetId(),
					ContainerName: containerNames[1],
					ImageID:       image.GetId(),
					ExePaths: []string{
						components[1].Executables[0].Path,
					},
				},
			},
			existingAcs: map[string]set.StringSet{
				componentsIDs[0]: set.NewStringSet(deployment.Containers[1].GetName()),
				componentsIDs[1]: set.NewStringSet(deployment.Containers[0].GetName()),
			},
			acsToUpdate: map[string]set.StringSet{
				componentsIDs[1]: set.NewStringSet(deployment.Containers[0].GetName(), deployment.Containers[1].GetName()),
			},
			acsToDelete: []string{componentsIDs[0]},
			imageChange: true,
		},
		{
			description: "Image change populate from database with removal request",
			updates: []*aggregatorPkg.ProcessUpdate{
				aggregatorPkg.NewProcessUpdate(image.GetId(), containerNames[0], set.NewStringSet(), aggregatorPkg.ToBeRemoved),
				aggregatorPkg.NewProcessUpdate(image.GetId(), containerNames[1], set.NewStringSet(), aggregatorPkg.FromDatabase),
			},
			indicators: map[string]indicatorModel{
				containerNames[0]: {
					DeploymentID:  deployment.GetId(),
					ContainerName: containerNames[0],
					ImageID:       image.GetId(),
					ExePaths: []string{
						components[1].Executables[0].Path,
						components[1].Executables[1].Path,
					},
				},
				containerNames[1]: {
					DeploymentID:  deployment.GetId(),
					ContainerName: containerNames[1],
					ImageID:       image.GetId(),
					ExePaths: []string{
						components[0].Executables[0].Path,
					},
				},
			},
			existingAcs: map[string]set.StringSet{
				componentsIDs[0]: set.NewStringSet(deployment.Containers[1].GetName()),
				componentsIDs[1]: set.NewStringSet(deployment.Containers[0].GetName()),
			},
			acsToUpdate: map[string]set.StringSet{
				componentsIDs[0]: set.NewStringSet(deployment.Containers[1].GetName()),
			},
			acsToDelete: []string{componentsIDs[1]},
			imageChange: true,
		},
		{
			description: "Update from cache adding new active contexts",
			updates: []*aggregatorPkg.ProcessUpdate{
				aggregatorPkg.NewProcessUpdate(image.GetId(), containerNames[0], set.NewStringSet(components[0].Executables[0].Path), aggregatorPkg.FromCache),
				aggregatorPkg.NewProcessUpdate(image.GetId(), containerNames[1], set.NewStringSet(components[1].Executables[0].Path, components[1].Executables[1].Path), aggregatorPkg.FromCache),
			},
			existingAcs: map[string]set.StringSet{
				componentsIDs[0]: set.NewStringSet(deployment.Containers[1].GetName()),
				componentsIDs[1]: set.NewStringSet(deployment.Containers[0].GetName()),
			},
			acsToUpdate: map[string]set.StringSet{
				componentsIDs[0]: set.NewStringSet(deployment.Containers[0].GetName(), deployment.Containers[1].GetName()),
				componentsIDs[1]: set.NewStringSet(deployment.Containers[0].GetName(), deployment.Containers[1].GetName()),
			},
		},
		{
			description: "update from cache no new change and no updates",
			updates: []*aggregatorPkg.ProcessUpdate{
				aggregatorPkg.NewProcessUpdate(image.GetId(), containerNames[0], set.NewStringSet(components[0].Executables[0].Path), aggregatorPkg.FromCache),
				aggregatorPkg.NewProcessUpdate(image.GetId(), containerNames[1], set.NewStringSet(components[0].Executables[1].Path, components[1].Executables[0].Path, components[1].Executables[1].Path), aggregatorPkg.FromCache),
			},
			existingAcs: map[string]set.StringSet{
				componentsIDs[0]: set.NewStringSet(deployment.Containers[0].GetName(), deployment.Containers[1].GetName()),
				componentsIDs[1]: set.NewStringSet(deployment.Containers[0].GetName(), deployment.Containers[1].GetName()),
			},
			acsToUpdate: map[string]set.StringSet{},
		},
		{
			description: "update from cache with removal request",
			updates: []*aggregatorPkg.ProcessUpdate{
				aggregatorPkg.NewProcessUpdate(image.GetId(), containerNames[0], set.NewStringSet(), aggregatorPkg.ToBeRemoved),
				aggregatorPkg.NewProcessUpdate(image.GetId(), containerNames[1], set.NewStringSet(components[0].Executables[1].Path), aggregatorPkg.FromCache),
			},
			// This should not be used in this test case.
			indicators: map[string]indicatorModel{
				containerNames[0]: {
					DeploymentID:  deployment.GetId(),
					ContainerName: containerNames[0],
					ImageID:       image.GetId(),
					ExePaths: []string{
						components[0].Executables[0].Path,
						components[1].Executables[1].Path,
					},
				},
				containerNames[1]: {
					DeploymentID:  deployment.GetId(),
					ContainerName: containerNames[1],
					ImageID:       image.GetId(),
					ExePaths: []string{
						components[0].Executables[0].Path,
						components[1].Executables[1].Path,
					},
				},
			},
			existingAcs: map[string]set.StringSet{
				componentsIDs[1]: set.NewStringSet(deployment.Containers[0].GetName()),
			},
			acsToUpdate: map[string]set.StringSet{
				componentsIDs[0]: set.NewStringSet(deployment.Containers[1].GetName()),
			},
			acsToDelete: []string{componentsIDs[1]},
		},
		{
			description: "First populate from database multiple component",
			updates: []*aggregatorPkg.ProcessUpdate{
				aggregatorPkg.NewProcessUpdate(image.GetId(), containerNames[0], set.NewStringSet(), aggregatorPkg.FromDatabase),
				aggregatorPkg.NewProcessUpdate(image.GetId(), containerNames[1], set.NewStringSet(), aggregatorPkg.FromDatabase),
			},
			indicators: map[string]indicatorModel{
				containerNames[0]: {
					DeploymentID:  deployment.GetId(),
					ContainerName: containerNames[0],
					ImageID:       image.GetId(),
					ExePaths: []string{
						components[0].Executables[2].Path,
					},
				},
				containerNames[1]: {
					DeploymentID:  deployment.GetId(),
					ContainerName: containerNames[1],
					ImageID:       image.GetId(),
					ExePaths: []string{
						components[1].Executables[2].Path,
					},
				},
			},
			acsToUpdate: map[string]set.StringSet{
				componentsIDs[0]: set.NewStringSet(deployment.Containers[0].GetName(), deployment.Containers[1].GetName()),
				componentsIDs[1]: set.NewStringSet(deployment.Containers[0].GetName(), deployment.Containers[1].GetName()),
			},
		},
		{
			description: "update from cache with removal request with multiple components",
			updates: []*aggregatorPkg.ProcessUpdate{
				aggregatorPkg.NewProcessUpdate(image.GetId(), containerNames[0], set.NewStringSet(), aggregatorPkg.ToBeRemoved),
				aggregatorPkg.NewProcessUpdate(image.GetId(), containerNames[1], set.NewStringSet(components[0].Executables[2].Path), aggregatorPkg.FromCache),
			},
			existingAcs: map[string]set.StringSet{
				componentsIDs[1]: set.NewStringSet(deployment.Containers[0].GetName()),
			},
			acsToUpdate: map[string]set.StringSet{
				componentsIDs[0]: set.NewStringSet(deployment.Containers[1].GetName()),
				componentsIDs[1]: set.NewStringSet(deployment.Containers[1].GetName()),
			},
		},
	}

	for _, testCase := range testCases {
		s.T().Run(testCase.description, func(t *testing.T) {
			s.mockAggregator.EXPECT().GetAndPrune(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
				func(_ func(string) bool, deploymentsSet set.StringSet) map[string][]*aggregatorPkg.ProcessUpdate {
					return map[string][]*aggregatorPkg.ProcessUpdate{
						deployment.GetId(): testCase.updates,
					}
				})
			var databaseFetchCount int
			for _, update := range testCase.updates {
				if update.FromDatabase() {
					databaseFetchCount++
				}
			}
			if databaseFetchCount > 0 {
				s.mockProcessIndicatorDataStore.EXPECT().SearchRawProcessIndicators(gomock.Any(), gomock.Any()).Times(databaseFetchCount).DoAndReturn(
					func(ctx context.Context, query *v1.Query) ([]*storage.ProcessIndicator, error) {
						queries := query.GetConjunction().Queries
						s.Assert().Len(queries, 2)
						var containerName string
						for _, q := range queries {
							mf := q.GetBaseQuery().GetMatchFieldQuery()

							switch mf.GetField() {
							case search.DeploymentID.String():
								assert.Equal(t, strconv.Quote(deployment.GetId()), mf.GetValue())
							case search.ContainerName.String():
								containerName = stripQuotes(mf.GetValue())
							default:
								s.Assert().Fail("unexpected query")
							}
						}

						var ret []*storage.ProcessIndicator

						for _, exec := range testCase.indicators[containerName].ExePaths {
							ret = append(ret, &storage.ProcessIndicator{
								Id:            uuid.NewV4().String(),
								ImageId:       testCase.indicators[containerName].ImageID,
								DeploymentId:  deployment.GetId(),
								ContainerName: containerName,
								Signal:        &storage.ProcessSignal{ExecFilePath: exec}},
							)
						}
						return ret, nil
					})
			}
			s.mockActiveComponentDataStore.EXPECT().SearchRawActiveComponents(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
				func(ctx context.Context, query *v1.Query) ([]*storage.ActiveComponent, error) {
					existingImageID := image.GetId()
					if testCase.imageChange {
						existingImageID = "something_else"
					}
					// Verify query
					assert.Equal(t, search.DeploymentID.String(), query.GetBaseQuery().GetMatchFieldQuery().GetField())
					assert.Equal(t, strconv.Quote(deployment.GetId()), query.GetBaseQuery().GetMatchFieldQuery().GetValue())
					var ret []*storage.ActiveComponent
					for componentID, containerNames := range testCase.existingAcs {
						acID := acConverter.ComposeID(deployment.GetId(), componentID)
						ac := &storage.ActiveComponent{
							Id:           acID,
							ComponentId:  componentID,
							DeploymentId: deployment.GetId(),
						}
						for containerName := range containerNames {
							ac.ActiveContextsSlice = append(ac.ActiveContextsSlice, &storage.ActiveComponent_ActiveContext{ContainerName: containerName, ImageId: existingImageID})
						}
						ret = append(ret, ac)
					}
					return ret, nil
				})
			s.mockActiveComponentDataStore.EXPECT().GetBatch(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
				func(ctx context.Context, ids []string) ([]*storage.ActiveComponent, error) {
					existingImageID := image.GetId()
					if testCase.imageChange {
						existingImageID = "something_else"
					}
					var ret []*storage.ActiveComponent
					requestedIds := set.NewStringSet(ids...)
					for componentID, containerNames := range testCase.existingAcs {
						acID := acConverter.ComposeID(deployment.GetId(), componentID)
						if !requestedIds.Contains(acID) {
							continue
						}
						ac := &storage.ActiveComponent{
							Id:           acID,
							ComponentId:  componentID,
							DeploymentId: deployment.GetId(),
						}
						for containerName := range containerNames {
							ac.ActiveContextsSlice = append(ac.ActiveContextsSlice, &storage.ActiveComponent_ActiveContext{
								ContainerName: containerName,
								ImageId:       existingImageID,
							})
						}
						ret = append(ret, ac)
					}
					return ret, nil
				})

			// Verify active components to be updated or deleted
			if len(testCase.acsToDelete) > 0 {
				s.mockActiveComponentDataStore.EXPECT().DeleteBatch(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
					func(ctx context.Context, ids ...string) error {
						expectedToDelete := set.NewStringSet()
						for _, componentID := range testCase.acsToDelete {
							expectedToDelete.Add(acConverter.ComposeID(deployment.GetId(), componentID))
						}
						assert.Equal(t, expectedToDelete, set.NewStringSet(ids...))
						return nil
					})
			}
			if len(testCase.acsToUpdate) > 0 {
				s.mockActiveComponentDataStore.EXPECT().UpsertBatch(gomock.Any(), gomock.Any()).Times(1).Return(nil).Do(func(_ context.Context, acs []*storage.ActiveComponent) {
					// Verify active components
					assert.Equal(t, len(testCase.acsToUpdate), len(acs))
					actualAcs := make(map[string]*storage.ActiveComponent, len(acs))
					for _, ac := range acs {
						_, _, err := acConverter.DecomposeID(ac.GetId())
						assert.NoError(t, err)
						actualAcs[ac.GetId()] = ac
					}

					for componentID, expectedContexts := range testCase.acsToUpdate {
						acID := acConverter.ComposeID(deployment.GetId(), componentID)
						assert.Contains(t, actualAcs, acID)
						assert.Equal(t, deployment.GetId(), actualAcs[acID].GetDeploymentId())
						assert.Equal(t, componentID, actualAcs[acID].GetComponentId())
						assert.Equal(t, acID, actualAcs[acID].GetId())
						assert.Equal(t, len(expectedContexts), len(actualAcs[acID].ActiveContextsSlice))
						for _, activeContext := range actualAcs[acID].ActiveContextsSlice {
							assert.Contains(t, expectedContexts, activeContext.GetContainerName())
							assert.Equal(t, image.GetId(), activeContext.GetImageId())
						}
					}
				})
			}
			updater.Update()
		})
	}
}

func stripQuotes(value string) string {
	return value[1 : len(value)-1]
}
