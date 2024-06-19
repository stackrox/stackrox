//go:build sql_integration

package service

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/golang/protobuf/jsonpb"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/random"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
)

const (
	namespace10pct = "Namespace10%"
	namepsace90pct = "Namespace90%"
)

func BenchmarkService_Export(b *testing.B) {
	_ = b
	testSuite := &servicePostgresTestSuiteInternals{}
	err := setupTest(b, testSuite)
	if err != nil {
		b.Error(err)
	}
	defer cleanupTest(b, testSuite)

	fmt.Println(time.Now().UTC().Unix(), "Injecting base images")
	baseImageIDs, baseImageNamesByIDs, err := injectInitialImages(testSuite)
	if err != nil {
		b.Error(err)
	}

	fmt.Println(time.Now().UTC().Unix(), "Injecting base deployments")
	err = injectRandomDeployments(testSuite, 500, baseImageIDs, baseImageNamesByIDs)
	if err != nil {
		b.Error(err)
	}

	fmt.Println(time.Now().UTC().Unix(), "Starting actual tests")
	b.Run("500", getServiceBenchmark(testSuite))

	fmt.Println(time.Now().UTC().Unix(), "Injecting 500 extra images")
	imageIDs := make([]string, 0, 20*len(baseImageIDs))
	imageIDs = append(imageIDs, baseImageIDs...)
	imageNamesByIDs := make(map[string]*storage.ImageName, 20*len(baseImageNamesByIDs))
	for k, v := range baseImageNamesByIDs {
		imageNamesByIDs[k] = v
	}
	newImageIDs, newImageNameByIDs, err := injectExtraImages(testSuite, 1)
	imageIDs = append(imageIDs, newImageIDs...)
	for k, v := range newImageNameByIDs {
		imageNamesByIDs[k] = v
	}

	fmt.Println(time.Now().UTC().Unix(), "Injecting 500 extra deployments")
	err = injectRandomDeployments(testSuite, 500, imageIDs, imageNamesByIDs)
	if err != nil {
		b.Error(err)
	}

	fmt.Println(time.Now().UTC().Unix(), "Continuing actual tests")
	b.Run("1000", getServiceBenchmark(testSuite))

	fmt.Println(time.Now().UTC().Unix(), "Injecting 4000 extra images")
	newImageIDs, newImageNameByIDs, err = injectExtraImages(testSuite, 8)
	imageIDs = append(imageIDs, newImageIDs...)
	for k, v := range newImageNameByIDs {
		imageNamesByIDs[k] = v
	}

	fmt.Println(time.Now().UTC().Unix(), "Injecting 4000 extra deployments")
	err = injectRandomDeployments(testSuite, 4000, imageIDs, imageNamesByIDs)
	if err != nil {
		b.Error(err)
	}

	fmt.Println(time.Now().UTC().Unix(), "Continuing actual tests (2)")
	b.Run("5 000", getServiceBenchmark(testSuite))

	fmt.Println(time.Now().UTC().Unix(), "Injecting 5000 extra images")
	newImageIDs, newImageNameByIDs, err = injectExtraImages(testSuite, 10)
	imageIDs = append(imageIDs, newImageIDs...)
	for k, v := range newImageNameByIDs {
		imageNamesByIDs[k] = v
	}

	fmt.Println(time.Now().UTC().Unix(), "Injecting 5000 extra deployments")
	err = injectRandomDeployments(testSuite, 5000, imageIDs, imageNamesByIDs)
	if err != nil {
		b.Error(err)
	}

	fmt.Println(time.Now().UTC().Unix(), "Continuing actual tests (3)")
	b.Run("10 000", getServiceBenchmark(testSuite))
}

func getBaseImageSet() ([]*storage.Image, error) {
	dataFile, err := os.Open("testdata/imgdata.json.gz")
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

/*
Sample run outcome.

Data injection phase (bench results removed from output)

BenchmarkService_Export
1718789959 Injecting base images
1718790077 Injecting base deployments
1718790082 Starting actual tests
... Results with a 500 item DB
1718790102 Injecting 500 extra images
1718790179 Injecting 500 extra deployments
1718790183 Continuing actual tests
... Results with a 1.000 item DB
1718790221 Injecting 4000 extra images
1718791150 Injecting 4000 extra deployments
1718791190 Continuing actual tests (2)
... Results with a 5.000 item DB
1718791574 Injecting 5000 extra images
1718792407 Injecting 5000 extra deployments
1718792448 Continuing actual tests (3)
... Results with a 10.000 item DB

Benchmark results

 DB Size        | No query         | Query 10% of db  | Query 90% of db  |
 (# deployment  |                  |                  |                  |
 and # images)  | Elapsed ns       | Elapsed ns       | Elapsed ns       |
----------------+------------------+------------------+------------------+
            500 |    9.165.222.465 |    1.896.871.287 |    9.100.526.550 |
----------------+------------------+------------------+------------------+
          1.000 |   17.562.401.077 |    3.695.793.796 |   16.655.958.011 |
----------------+------------------+------------------+------------------+
          5.000 |  232.410.473.350 |   17.445.117.452 |  133.301.396.811 |
----------------+------------------+------------------+------------------+
         10.000 |  385.138.425.396 |   35.631.981.696 |  302.740.800.784 |
----------------+------------------+------------------+------------------+

The structure of the service call is:
- Walk by query on deployments
  - pull elements directly from cache if empty query
  - pull elements from DB otherwise
  [ for each deployment :
    - for each container in the deployment:
      - identify the container image ID
      - get the container image from cache
      - on cache miss, get the container image from DB
        - get the image metadata from DB (table images)
        - get ImageCVEEdge objects (table image_cve_edges)
        - get CVE IDs from the edge objects
        - get ImageComponentEdge objects (table image_component_edges)
        - get component IDs from edge objects
        - get ImageComponent objects (table image_components)
        - get (Image)ComponentCVEEdge objects (table image_component_cve_edges)
        - get ImageCVE objects (table image_cves)
        - reconstruct image object from fetched elements
  ]
*/

func injectInitialImages(suite *servicePostgresTestSuiteInternals) ([]string, map[string]*storage.ImageName, error) {
	images, err := getBaseImageSet()
	if err != nil {
		return nil, nil, err
	}

	imageIDs := make([]string, 0, len(images))
	imageNamesByIDs := make(map[string]*storage.ImageName, len(images))
	allAccessCtx := sac.WithAllAccess(suite.ctx)

	for _, img := range images {
		imgID := img.GetId()
		imageIDs = append(imageIDs, imgID)
		imageNamesByIDs[imgID] = img.GetName()
		err = suite.images.UpsertImage(allAccessCtx, img)
		if err != nil {
			return nil, nil, err
		}
	}

	return imageIDs, imageNamesByIDs, nil
}

func injectExtraImages(suite *servicePostgresTestSuiteInternals, copyCount int) ([]string, map[string]*storage.ImageName, error) {
	images, err := getBaseImageSet()
	if err != nil {
		return nil, nil, err
	}

	imageIDs := make([]string, 0, len(images))
	imageNamesByIDs := make(map[string]*storage.ImageName, len(images))
	allAccessCtx := sac.WithAllAccess(suite.ctx)

	for _, img := range images {
		imgName := img.GetName()
		for i := 0; i < copyCount; i++ {
			clone := img.Clone()
			clone.Id, err = random.GenerateString(65, random.HexValues)
			cloneID := clone.GetId()
			imageIDs = append(imageIDs, cloneID)
			imageNamesByIDs[cloneID] = imgName
			err = suite.images.UpsertImage(allAccessCtx, clone)
			if err != nil {
				return nil, nil, err
			}
		}
	}

	return imageIDs, imageNamesByIDs, nil
}

func injectRandomDeployments(
	suite *servicePostgresTestSuiteInternals,
	count int,
	imageIDs []string,
	imageNamesByIDs map[string]*storage.ImageName,
) error {
	baseContainer := &storage.Container{}
	err := testutils.FullInit(baseContainer, testutils.UniqueInitializer(), testutils.JSONFieldsFilter)
	if err != nil {
		return err
	}
	for i := 0; i < count; i++ {
		d := &storage.Deployment{}
		err = testutils.FullInit(d, testutils.UniqueInitializer(), testutils.JSONFieldsFilter)
		if err != nil {
			return err
		}
		nContainers := i%3 + 1
		containers := make([]*storage.Container, 0, 3)
		for j := 0; j < nContainers; j++ {
			ix := int(rand.Int31()) % len(imageIDs)
			imgID := imageIDs[ix]
			imgName := imageNamesByIDs[imgID]
			containerImage := &storage.ContainerImage{
				Id:             imgID,
				Name:           imgName,
				NotPullable:    false,
				IsClusterLocal: false,
			}
			container := baseContainer.Clone()
			container.Image = containerImage
			containers = append(containers, container)
		}
		if i%10 == 9 {
			d.Namespace = namespace10pct
		} else {
			d.Namespace = namepsace90pct
		}
		d.Containers = containers
		ctx := sac.WithAllAccess(context.Background())
		err := suite.deployments.UpsertDeployment(ctx, d)
		if err != nil {
			return err
		}
	}
	return nil
}

func getServiceBenchmark(suite *servicePostgresTestSuiteInternals) func(b *testing.B) {
	return func(b *testing.B) {
		testScenarios := []struct {
			name            string
			query           string
			targetNamespace string
		}{
			{
				name: "No Query",
			},
			{
				name:            "Query 10% of dataset",
				targetNamespace: namespace10pct,
			},
			{
				name:            "Query 90% of dataset",
				targetNamespace: namepsace90pct,
			},
		}

		for _, scenario := range testScenarios {
			b.Run(scenario.name, func(b *testing.B) {
				request := &v1.VulnMgmtExportWorkloadsRequest{Timeout: 3600}
				if scenario.targetNamespace != "" {
					request.Query = fmt.Sprintf("Namespace:%s", scenario.targetNamespace)
				}
				conn, closeFunc, err := createGRPCWorkloadsService(suite)
				if err != nil {
					b.Error(err)
				}
				defer closeFunc()

				client := v1.NewVulnMgmtServiceClient(conn)
				for i := 0; i < b.N; i++ {
					_, err = receiveWorkloads(suite.ctx, client, request, true)
					if err != nil {
						b.Error(err)
					}
				}
			})
		}
	}
}
