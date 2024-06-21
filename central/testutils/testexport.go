package testutils

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/jsonpb"
	grpcMiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/random"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const (
	namespace10pct = "Namespace10%"
	namepsace90pct = "Namespace90%"
)

// ExportServicePostgresTestHelper is a utility to help testing the
// export APIs (takes over the data injection).
type ExportServicePostgresTestHelper struct {
	Ctx         context.Context
	pool        *pgtest.TestPostgres
	Deployments deploymentDataStore.DataStore
	Images      imageDataStore.DataStore
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
	h.Images = imageDataStore.GetTestPostgresDataStore(tb, h.pool)
	return nil
}

// TearDownTest cleans up the ExportServicePostgresTestHelper resources after testing.
func (h *ExportServicePostgresTestHelper) TearDownTest(tb testing.TB) {
	h.pool.Teardown(tb)
	h.pool.Close()
}

// getAuthInterceptor returns a GRPC Steam Interceptor that overrides the server context to make sure all calls are authenticated.
func getAuthInterceptor(ctx context.Context) func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, &grpcMiddleware.WrappedServerStream{
			ServerStream:   ss,
			WrappedContext: ctx,
		})
	}
}

// CreateGRPCStreamingService creates a streaming server, registers the target
// services there, and returns a connection to the streaming server along with
// a function to close the connection.
func (h *ExportServicePostgresTestHelper) CreateGRPCStreamingService(
	_ testing.TB,
	registerServices func(registrar grpc.ServiceRegistrar),
) (*grpc.ClientConn, func(), error) {
	bufferSize := 1024 * 1024
	listener := bufconn.Listen(bufferSize)

	server := grpc.NewServer(grpc.StreamInterceptor(getAuthInterceptor(h.Ctx)))
	registerServices(server)

	go func() {
		utils.IgnoreError(func() error { return server.Serve(listener) })
	}()

	conn, err := grpc.DialContext(h.Ctx, "",
		grpc.WithContextDialer(
			func(ctx context.Context, _ string) (net.Conn, error) {
				return listener.DialContext(ctx)
			},
		),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, nil, err
	}

	closeFunc := func() {
		utils.IgnoreError(listener.Close)
		server.Stop()
	}
	return conn, closeFunc, nil
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

// GetBaseImageSet returns a set of realistic images for testing purposes.
func GetBaseImageSet() ([]*storage.Image, error) {
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
	unmarshaler := jsonpb.Unmarshaler{}

	images := make([]*storage.Image, 0, 500)

	for jsonReader.More() {
		img := &storage.Image{}
		err = unmarshaler.UnmarshalNext(jsonReader, img)
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
	baseImages, err := GetBaseImageSet()
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
			clone := img.Clone()
			hash, err := random.GenerateString(64, random.HexValues)
			if err != nil {
				return nil, nil, err
			}
			clone.Id = fmt.Sprintf("sha256:%s", hash)
			err = h.Images.UpsertImage(upsertCtx, clone)
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
) error {
	upsertCtx := sac.WithAllAccess(context.Background())
	for i := 0; i < count; i++ {
		deployment := &storage.Deployment{}
		err := testutils.FullInit(deployment, testutils.UniqueInitializer(), testutils.JSONFieldsFilter)
		if err != nil {
			return err
		}
		nbContainers := (i % 3) + 1
		containers := make([]*storage.Container, 0, nbContainers)
		for j := 0; j < nbContainers; j++ {
			container := &storage.Container{}
			err := testutils.FullInit(container, testutils.UniqueInitializer(), testutils.JSONFieldsFilter)
			if err != nil {
				return err
			}
			idx := int(rand.Int31()) % len(imageIDs)
			imageID := imageIDs[idx]
			imageName := imageNamesByIDs[imageID]
			containerImage := &storage.ContainerImage{
				Id:             imageID,
				Name:           imageName,
				NotPullable:    false,
				IsClusterLocal: false,
			}
			container.Image = containerImage
			containers = append(containers, container)
		}
		deployment.Containers = containers
		if i%10 == 9 {
			deployment.Namespace = namespace10pct
		} else {
			deployment.Namespace = namepsace90pct
		}
		err = h.Deployments.UpsertDeployment(upsertCtx, deployment)
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
	datasetSizes := []int{500}
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
		images, err := GetBaseImageSet()
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
			fmt.Println(time.Now().UTC().Unix(), "Injecting", delta, "images")
			addedImageIDs, addedImageNamesByID, err := h.InjectImages(b, delta)
			if err != nil {
				b.Error(err)
			}
			imageIDs = append(imageIDs, addedImageIDs...)
			for imageID, imageName := range addedImageNamesByID {
				imageNamesByIDs[imageID] = imageName
			}
		}
		fmt.Println(time.Now().UTC().Unix(), "Injecting", delta, "deployments")
		err := h.InjectDeployments(b, delta, imageIDs, imageNamesByIDs)
		if err != nil {
			b.Error(err)
		}
		fmt.Println(time.Now().UTC().Unix(), "Test iteration", ix+1)
		b.Run(fmt.Sprintf("%d", datasetSize), benchmark)
	}

}

// TestScenario contains the parameters for an export API test.
type TestScenario struct {
	Name            string
	TargetNamespace string
}

// GetBaseTestCases returns a basic set of TestScenario objects.
func GetBaseTestCases() []TestScenario {
	return []TestScenario{
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
