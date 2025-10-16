package testutils

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"slices"
	"strings"
	"testing"

	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	podDatastore "github.com/stackrox/rox/central/pod/datastore"
	deploymentsView "github.com/stackrox/rox/central/views/deployments"
	imagesView "github.com/stackrox/rox/central/views/images"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/random"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/testutils"
)

const (
	namespace10pct = "Namespace10%"
	namepsace90pct = "Namespace90%"
)

var (
	log = logging.LoggerForModule()
)

// ExportServicePostgresTestHelper is a utility to help testing the
// export APIs (takes over the data injection).
type ExportServicePostgresTestHelper struct {
	Ctx            context.Context
	pool           *pgtest.TestPostgres
	Deployments    deploymentDataStore.DataStore
	Images         imageDataStore.DataStore
	ImageView      imagesView.ImageView
	Pods           podDatastore.DataStore
	DeploymentView deploymentsView.DeploymentView
}

// SetupTest prepares the ExportServicePostgresTestHelper struct for testing.
func (h *ExportServicePostgresTestHelper) SetupTest(tb testing.TB) error {
	h.Ctx = sac.WithGlobalAccessScopeChecker(
		context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Deployment, resources.Image),
		),
	)
	h.pool = pgtest.ForT(tb)
	deploymentStore, err := deploymentDataStore.GetTestPostgresDataStore(tb, h.pool)
	if err != nil {
		return err
	}
	h.Deployments = deploymentStore
	h.DeploymentView = deploymentsView.NewDeploymentView(h.pool)
	h.Images = imageDataStore.GetTestPostgresDataStore(tb, h.pool)
	h.ImageView = imagesView.NewImageView(h.pool)
	h.Pods = podDatastore.GetTestPostgresDataStore(tb, h.pool)
	return nil
}

func getImageSetPath() (string, error) {
	// Go up the directory tree from the current working directory
	// to location where the subtree to the image data file matches.
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	pathElems := strings.Split(cwd, string(os.PathSeparator))
	for i := len(pathElems); i >= 0; i-- {
		basePath := strings.Join(pathElems[:i], string(os.PathSeparator))
		imageDataPathElems := []string{basePath, "central", "testutils", "testdata", "imgdata.json.gz"}
		imageDataPath := strings.Join(imageDataPathElems, string(os.PathSeparator))
		_, err := os.Stat(imageDataPath)
		if err == nil {
			return imageDataPath, nil
		}
		if os.IsNotExist(err) {
			continue
		}
		return "", err
	}
	return "", nil
}

// getTestImages returns a set of realistic images for testing purposes.
func getTestImages() ([]*storage.Image, error) {
	imageDataPath, err := getImageSetPath()
	if err != nil {
		return nil, err
	}
	dataFile, err := os.Open(imageDataPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = dataFile.Close() }()

	zipReader, err := gzip.NewReader(dataFile)
	if err != nil {
		return nil, err
	}
	defer func() { _ = zipReader.Close() }()

	jsonReader := json.NewDecoder(zipReader)
	unmarshaler := jsonutil.JSONUnmarshaler()

	images := make([]*storage.Image, 0, 500)

	for jsonReader.More() {
		b := json.RawMessage{}
		err := jsonReader.Decode(&b)
		if err != nil {
			return nil, err
		}
		img := &storage.Image{}
		err = unmarshaler.Unmarshal(b, img)
		if err != nil {
			return nil, err
		}
		images = append(images, img)
	}

	return images, nil
}

// InjectImages creates a set of images in DB with random identifiers
// from a pool of realistic images.
func (h *ExportServicePostgresTestHelper) InjectImages(
	_ testing.TB,
	count int,
) ([]string, map[string]*storage.ImageName, error) {
	baseImages, err := getTestImages()
	if err != nil {
		return nil, nil, err
	}

	imageIDs := make([]string, 0, count)
	imageNamesByID := make(map[string]*storage.ImageName, count)
	copies := count / len(baseImages)
	extras := count % len(baseImages)
	upsertCtx := sac.WithAllAccess(context.Background())
	for i := 0; i < len(baseImages); i++ {
		copyCount := copies
		if i < extras {
			copyCount++
		}
		img := baseImages[i]
		imgName := img.GetName()
		for j := 0; j < copyCount; j++ {
			clone := img.CloneVT()
			hash := random.GenerateString(64, random.HexValues)
			clone.SetId(fmt.Sprintf("sha256:%s", hash))
			err := h.Images.UpsertImage(upsertCtx, clone)
			if err != nil {
				return nil, nil, err
			}
			cloneID := clone.GetId()
			imageIDs = append(imageIDs, cloneID)
			imageNamesByID[cloneID] = imgName
		}
	}
	return imageIDs, imageNamesByID, nil
}

// InjectDeployments creates a set of pseudo-random deployments in DB
func (h *ExportServicePostgresTestHelper) InjectDeployments(
	_ testing.TB,
	count int,
	imageIDs []string,
	imageNamesByIDs map[string]*storage.ImageName,
) ([]*storage.Deployment, error) {
	upsertCtx := sac.WithAllAccess(context.Background())
	deployments := make([]*storage.Deployment, 0, count)
	for i := 0; i < count; i++ {
		deployment := &storage.Deployment{}
		err := testutils.FullInit(deployment, testutils.UniqueInitializer(), testutils.JSONFieldsFilter)
		if err != nil {
			return nil, err
		}
		nbContainers := (i % 3) + 1
		containers := make([]*storage.Container, 0, nbContainers)
		for j := 0; j < nbContainers; j++ {
			container := &storage.Container{}
			err := testutils.FullInit(container, testutils.UniqueInitializer(), testutils.JSONFieldsFilter)
			if err != nil {
				return nil, err
			}
			idx := int(rand.Int31()) % len(imageIDs)
			imageID := imageIDs[idx]
			imageName := imageNamesByIDs[imageID]
			containerImage := &storage.ContainerImage{}
			containerImage.SetId(imageID)
			containerImage.SetName(imageName)
			containerImage.SetNotPullable(false)
			containerImage.SetIsClusterLocal(false)
			// region ensure enum values are valid
			// Note: the unique initializer considers enum fields as int32
			// and fills them with values that are mostly out of the valid
			// range. These get reverted to fixed valid values so decoders
			// do not break.
			container.SetImage(containerImage)
			for _, v := range container.GetVolumes() {
				v.SetMountPropagation(storage.Volume_NONE)
			}
			if container.GetConfig() != nil {
				for _, e := range container.GetConfig().GetEnv() {
					e.SetEnvVarSource(storage.ContainerConfig_EnvironmentConfig_UNKNOWN)
				}
			}
			for _, portConfig := range container.GetPorts() {
				portConfig.SetExposure(storage.PortConfig_INTERNAL)
				for _, exposureInfo := range portConfig.GetExposureInfos() {
					exposureInfo.SetLevel(storage.PortConfig_INTERNAL)
				}
			}
			// endregion ensure enum values are valid
			containers = append(containers, container)
		}
		deployment.SetContainers(containers)
		if i%10 == 9 {
			deployment.SetNamespace(namespace10pct)
		} else {
			deployment.SetNamespace(namepsace90pct)
		}
		// Set the enum values to valid data.
		for _, portConfig := range deployment.GetPorts() {
			portConfig.SetExposure(storage.PortConfig_INTERNAL)
			for _, exposureInfo := range portConfig.GetExposureInfos() {
				exposureInfo.SetLevel(storage.PortConfig_INTERNAL)
			}
		}

		err = h.Deployments.UpsertDeployment(upsertCtx, deployment)
		if err != nil {
			return nil, err
		}
		deployments = append(deployments, deployment)
	}
	return deployments, nil
}

// InjectPods creates a set of pseudo-random pods in DB
func (h *ExportServicePostgresTestHelper) InjectPods(
	_ testing.TB,
	deployments []*storage.Deployment,
) error {
	upsertCtx := sac.WithAllAccess(context.Background())
	for _, dep := range deployments {
		pod := &storage.Pod{}
		err := testutils.FullInit(pod, testutils.UniqueInitializer(), testutils.JSONFieldsFilter)
		if err != nil {
			return err
		}
		pod.SetDeploymentId(dep.GetId())
		pod.SetNamespace(dep.GetNamespace())
		pod.SetClusterId(dep.GetClusterId())

		podLiveInstances := make([]*storage.ContainerInstance, 0, len(dep.GetContainers()))
		for _, container := range dep.GetContainers() {
			instance := &storage.ContainerInstance{}
			err := testutils.FullInit(instance, testutils.UniqueInitializer(), testutils.JSONFieldsFilter)
			if err != nil {
				return err
			}

			instance.SetContainerName(container.GetName())
			instance.SetContainingPodId(pod.GetId())
			instance.SetImageDigest(container.GetImage().GetId())

			podLiveInstances = append(podLiveInstances, instance)
		}
		pod.SetLiveInstances(podLiveInstances)

		err = h.Pods.UpsertPod(upsertCtx, pod)
		if err != nil {
			return err
		}
	}
	return nil
}

// InjectDataAndRunBenchmark pushes datasets of various sizes to database,
// and runs the provided benchmark function against them.
func (h *ExportServicePostgresTestHelper) InjectDataAndRunBenchmark(
	b *testing.B,
	injectImages bool,
	benchmark func(b *testing.B),
) {
	// For the standard go benchmark tests, have minimal scale to ensure
	// the test runs properly.
	datasetSizes := []int{10}
	// The test runs by default with a lower scale as smoke test
	// in the benchmark unit tests. To test at higher scales (takes time),
	// run the test with ROX_SCALE_TEST set to a non-empty value
	// in the test environment.
	scale := os.Getenv("ROX_SCALE_TEST")
	if scale != "" {
		datasetSizes = []int{500, 1000, 2000, 5000, 10000}
	}
	imageIDs := make([]string, 0)
	imageNamesByIDs := make(map[string]*storage.ImageName)
	if !injectImages {
		images, err := getTestImages()
		if err != nil {
			b.Error(err)
		}
		for _, image := range images {
			imageID := image.GetId()
			imageName := image.GetName()
			imageIDs = append(imageIDs, imageID)
			imageNamesByIDs[imageID] = imageName
		}
	}

	slices.Sort(datasetSizes)
	lastDatasetSize := 0
	for ix, datasetSize := range datasetSizes {
		delta := datasetSize - lastDatasetSize
		if injectImages {
			log.Info("Injecting ", delta, " images")
			addedImageIDs, addedImageNamesByID, err := h.InjectImages(b, delta)
			if err != nil {
				b.Error(err)
			}
			imageIDs = append(imageIDs, addedImageIDs...)
			for imageID, imageName := range addedImageNamesByID {
				imageNamesByIDs[imageID] = imageName
			}
		}
		log.Info("Injecting ", delta, " deployments")
		deployments, err := h.InjectDeployments(b, delta, imageIDs, imageNamesByIDs)
		if err != nil {
			b.Error(err)
		}
		err = h.InjectPods(b, deployments)
		if err != nil {
			b.Error(err)
		}
		log.Info("Test iteration ", ix+1)
		b.Run(fmt.Sprintf("%d", datasetSize), benchmark)
		lastDatasetSize = datasetSize
	}
}

// ExportTestCase contains the parameters for an export API test.
type ExportTestCase struct {
	Name            string
	TargetNamespace string
}

// GetExportTestCases returns a minimal list of TestScenario objects.
func GetExportTestCases() []ExportTestCase {
	return []ExportTestCase{
		{
			Name: "No Query",
		},
		{
			Name:            "Query 10% of dataset",
			TargetNamespace: namespace10pct,
		},
		{
			Name:            "Query 90% of dataset",
			TargetNamespace: namepsace90pct,
		},
	}
}
