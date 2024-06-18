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

	fmt.Println(time.Now().UTC().Unix(), "Injecting 9000 extra images")
	newImageIDs, newImageNameByIDs, err = injectExtraImages(testSuite, 18)
	imageIDs = append(imageIDs, newImageIDs...)
	for k, v := range newImageNameByIDs {
		imageNamesByIDs[k] = v
	}

	fmt.Println(time.Now().UTC().Unix(), "Injecting 9000 extra deployments")
	err = injectRandomDeployments(testSuite, 9000, imageIDs, imageNamesByIDs)
	if err != nil {
		b.Error(err)
	}

	fmt.Println(time.Now().UTC().Unix(), "Continuing actual tests (2)")
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
