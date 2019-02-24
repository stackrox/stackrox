package tests

import (
	"archive/zip"
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	deploymentStore "github.com/stackrox/rox/central/deployment/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This doesn't actually check for the existence of a specific deployment, but any deployment
func checkDeploymentExists(t *testing.T) {
	conn, err := grpcConnection()
	require.NoError(t, err)

	service := v1.NewDeploymentServiceClient(conn)

	// 10 seconds should be enough for a deployment to be returned
	for i := 0; i < 5; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		resp, err := service.ListDeployments(ctx, &v1.RawQuery{})
		cancel()
		require.NoError(t, err)
		if len(resp.Deployments) != 0 {
			return
		}
		time.Sleep(2 * time.Second)
	}
	assert.Fail(t, "Failed to find a deployment")
}

// Grab the backup DB and open it, ensuring that there are values for deployments
func TestBackup(t *testing.T) {
	setupNginxLatestTagDeployment(t)
	defer teardownNginxLatestTagDeployment(t)

	checkDeploymentExists(t)

	out, err := os.Create("backup.zip")
	require.NoError(t, err)
	defer func() {
		_ = os.Remove("backup.zip")
	}()

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	req, err := http.NewRequest(http.MethodGet, "https://"+apiEndpoint+"/db/backup", nil)
	require.NoError(t, err)
	if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer utils.IgnoreError(resp.Body.Close)
	_, err = io.Copy(out, resp.Body)
	require.NoError(t, err)

	defer utils.IgnoreError(out.Close)

	zipFile, err := zip.OpenReader("backup.zip")
	require.NoError(t, err)
	defer utils.IgnoreError(zipFile.Close)

	var boltFileEntry *zip.File
	for _, f := range zipFile.File {
		if f.Name == "bolt.db" {
			boltFileEntry = f
			break
		}
	}
	require.NotNil(t, boltFileEntry)

	boltFile, err := boltFileEntry.Open()
	require.NoError(t, err)

	boltOut, err := os.Create("backup.db")
	require.NoError(t, err)
	defer func() {
		_ = os.Remove("backup.db")
	}()

	_, err = io.Copy(boltOut, boltFile)
	require.NoError(t, err)
	require.NoError(t, boltFile.Close())
	require.NoError(t, boltOut.Close())

	b, err := bolthelper.New("backup.db")
	require.NoError(t, err)

	depStore, err := deploymentStore.New(b)
	require.NoError(t, err)
	deployments, err := depStore.GetDeployments()
	require.NoError(t, err)
	assert.NotEmpty(t, deployments)
}
