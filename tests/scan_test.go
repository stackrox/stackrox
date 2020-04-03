package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func verifySummariesExist(t *testing.T, image *storage.Image, shouldExist bool) {
	assertFunc := assert.NotEmpty
	if !shouldExist {
		assertFunc = assert.Empty
	}

	var checkedAtLeastOnce bool
	for _, component := range image.GetScan().GetComponents() {
		for _, vuln := range component.GetVulns() {
			checkedAtLeastOnce = true
			assertFunc(t, vuln.Summary)
		}
	}
	// Ensure there are components and vulns
	assert.True(t, checkedAtLeastOnce)
}

// Grab the backup DB and open it, ensuring that there are values for deployments
func TestScan(t *testing.T) {
	deployment := "nginx-1-10"
	imageID := "sha256:6202beb06ea61f44179e02ca965e8e13b961d12640101fca213efbfd145d7575"
	image := fmt.Sprintf("docker.io/library/nginx:1.10@%s", imageID)
	setupDeployment(t, image, deployment)
	defer teardownDeployment(t, deployment)

	conn := testutils.GRPCConnectionToCentral(t)
	imageService := v1.NewImageServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	var resp *storage.Image
	var err error
	err = retry.WithRetry(func() error {
		resp, err = imageService.GetImage(ctx, &v1.ResourceByID{
			Id: imageID,
		})
		if err != nil {
			return retry.MakeRetryable(err)
		}
		return nil
	}, retry.OnFailedAttempts(func(err error) {
		log.Errorf("error getting image: %v", err)
		time.Sleep(5 * time.Second)
	}), retry.Tries(20))
	require.NoError(t, err)

	resp, err = imageService.GetImage(ctx, &v1.ResourceByID{
		Id: imageID,
	})
	require.NoError(t, err)
	verifySummariesExist(t, resp, true)

	resp, err = imageService.ScanImage(ctx, &v1.ScanImageRequest{
		ImageName: image,
	})
	require.NoError(t, err)
	verifySummariesExist(t, resp, true)

	resp, err = imageService.GetImage(ctx, &v1.ResourceByID{
		Id: resp.GetId(),
	})
	require.NoError(t, err)
	verifySummariesExist(t, resp, true)

	resp, err = imageService.ScanImage(ctx, &v1.ScanImageRequest{
		ImageName: "docker.io/library/nginx:1.10",
		Force:     true,
	})
	require.NoError(t, err)
	verifySummariesExist(t, resp, true)

	resp, err = imageService.GetImage(ctx, &v1.ResourceByID{
		Id: resp.GetId(),
	})
	require.NoError(t, err)
	verifySummariesExist(t, resp, true)
}
