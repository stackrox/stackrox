package splunk

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	imageMocks "github.com/stackrox/rox/central/image/datastore/mocks"
	imageV2Mocks "github.com/stackrox/rox/central/imagev2/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestVulnMgmtHandler_V1(t *testing.T) {
	t.Setenv(features.FlattenImageData.EnvVar(), "false")

	const (
		deploymentID = "dep-1"
		imageDigest  = "sha256:abc123"
		imageUUID    = "uuid-v2-001"
	)

	ctrl := gomock.NewController(t)
	deployments := deploymentMocks.NewMockDataStore(ctrl)
	images := imageMocks.NewMockDataStore(ctrl)
	imagesV2 := imageV2Mocks.NewMockDataStore(ctrl)

	deployments.EXPECT().GetDeploymentIDs(gomock.Any()).Return([]string{deploymentID}, nil)
	deployments.EXPECT().GetDeployment(gomock.Any(), deploymentID).Return(&storage.Deployment{
		Id:          deploymentID,
		Name:        "my-deploy",
		ClusterName: "cluster-1",
		Namespace:   "default",
		Containers: []*storage.Container{
			{
				Image: &storage.ContainerImage{Id: imageDigest, IdV2: imageUUID},
			},
		},
	}, true, nil)

	// V1 path: lookup by digest, NOT by UUID.
	images.EXPECT().GetImage(gomock.Any(), imageDigest).Return(&storage.Image{
		Id: imageDigest,
		Name: &storage.ImageName{
			Registry: "docker.io",
			Remote:   "library/nginx",
			Tag:      "latest",
		},
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{Created: timestamppb.Now()},
		},
		Scan: &storage.ImageScan{
			OperatingSystem: "linux",
			Components: []*storage.EmbeddedImageScanComponent{
				{
					Name:    "openssl",
					Version: "1.1.1",
					Vulns: []*storage.EmbeddedVulnerability{
						{Cve: "CVE-2021-0001", Cvss: 7.5, SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.1.2"}},
					},
				},
			},
		},
	}, true, nil)

	handler := NewVulnMgmtHandler(deployments, images, imagesV2)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusOK, rr.Code)

	var events []json.RawMessage
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &events))
	require.Len(t, events, 3) // deployment + image + cve

	// Verify deployment event uses SHA digest.
	var depEvent splunkDeploymentEvent
	require.NoError(t, json.Unmarshal(events[0], &depEvent))
	assert.Equal(t, "deployment", depEvent.Type)
	assert.Equal(t, imageDigest, depEvent.ImageDigest)

	// Verify image event uses SHA digest.
	var imgEvent splunkImageEvent
	require.NoError(t, json.Unmarshal(events[1], &imgEvent))
	assert.Equal(t, "image", imgEvent.Type)
	assert.Equal(t, imageDigest, imgEvent.ImageDigest)

	// Verify CVE event uses SHA digest.
	var cveEvent splunkCVEEvent
	require.NoError(t, json.Unmarshal(events[2], &cveEvent))
	assert.Equal(t, "cve", cveEvent.Type)
	assert.Equal(t, imageDigest, cveEvent.ImageDigest)
	assert.Equal(t, "CVE-2021-0001", cveEvent.CVE)
}

func TestVulnMgmtHandler_V2(t *testing.T) {
	t.Setenv(features.FlattenImageData.EnvVar(), "true")

	const (
		deploymentID = "dep-1"
		imageDigest  = "sha256:abc123"
		imageUUID    = "uuid-v2-001"
	)

	ctrl := gomock.NewController(t)
	deployments := deploymentMocks.NewMockDataStore(ctrl)
	images := imageMocks.NewMockDataStore(ctrl)
	imagesV2 := imageV2Mocks.NewMockDataStore(ctrl)

	deployments.EXPECT().GetDeploymentIDs(gomock.Any()).Return([]string{deploymentID}, nil)
	deployments.EXPECT().GetDeployment(gomock.Any(), deploymentID).Return(&storage.Deployment{
		Id:          deploymentID,
		Name:        "my-deploy",
		ClusterName: "cluster-1",
		Namespace:   "default",
		Containers: []*storage.Container{
			{
				Image: &storage.ContainerImage{Id: imageDigest, IdV2: imageUUID},
			},
		},
	}, true, nil)

	// V2 path: lookup by UUID, NOT by digest.
	imagesV2.EXPECT().GetImage(gomock.Any(), imageUUID).Return(&storage.ImageV2{
		Id:     imageUUID,
		Digest: imageDigest,
		Name: &storage.ImageName{
			Registry: "docker.io",
			Remote:   "library/nginx",
			Tag:      "latest",
		},
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{Created: timestamppb.Now()},
		},
		Scan: &storage.ImageScan{
			OperatingSystem: "linux",
			Components: []*storage.EmbeddedImageScanComponent{
				{
					Name:    "openssl",
					Version: "1.1.1",
					Vulns: []*storage.EmbeddedVulnerability{
						{Cve: "CVE-2021-0001", Cvss: 7.5, SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.1.2"}},
					},
				},
			},
		},
	}, true, nil)

	handler := NewVulnMgmtHandler(deployments, images, imagesV2)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusOK, rr.Code)

	var events []json.RawMessage
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &events))
	require.Len(t, events, 3) // deployment + image + cve

	// Verify deployment event uses SHA digest, NOT UUID.
	var depEvent splunkDeploymentEvent
	require.NoError(t, json.Unmarshal(events[0], &depEvent))
	assert.Equal(t, "deployment", depEvent.Type)
	assert.Equal(t, imageDigest, depEvent.ImageDigest)

	// Verify image event uses SHA digest, NOT UUID.
	var imgEvent splunkImageEvent
	require.NoError(t, json.Unmarshal(events[1], &imgEvent))
	assert.Equal(t, "image", imgEvent.Type)
	assert.Equal(t, imageDigest, imgEvent.ImageDigest)

	// Verify CVE event uses SHA digest, NOT UUID.
	var cveEvent splunkCVEEvent
	require.NoError(t, json.Unmarshal(events[2], &cveEvent))
	assert.Equal(t, "cve", cveEvent.Type)
	assert.Equal(t, imageDigest, cveEvent.ImageDigest)
	assert.Equal(t, "CVE-2021-0001", cveEvent.CVE)
}

func TestVulnMgmtHandler_V2SkipsEmptyUUID(t *testing.T) {
	t.Setenv(features.FlattenImageData.EnvVar(), "true")

	const (
		deploymentID = "dep-1"
		imageDigest  = "sha256:abc123"
	)

	ctrl := gomock.NewController(t)
	deployments := deploymentMocks.NewMockDataStore(ctrl)
	images := imageMocks.NewMockDataStore(ctrl)
	imagesV2 := imageV2Mocks.NewMockDataStore(ctrl)

	deployments.EXPECT().GetDeploymentIDs(gomock.Any()).Return([]string{deploymentID}, nil)
	deployments.EXPECT().GetDeployment(gomock.Any(), deploymentID).Return(&storage.Deployment{
		Id:          deploymentID,
		Name:        "my-deploy",
		ClusterName: "cluster-1",
		Namespace:   "default",
		Containers: []*storage.Container{
			{
				// IdV2 is empty — this container should be skipped entirely.
				Image: &storage.ContainerImage{Id: imageDigest},
			},
		},
	}, true, nil)

	handler := NewVulnMgmtHandler(deployments, images, imagesV2)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusOK, rr.Code)

	var events []json.RawMessage
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &events))
	// No deployment or image events because the only container has empty IdV2.
	assert.Empty(t, events)
}
