package tests

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"bitbucket.org/stack-rox/apollo/central/db/boltdb"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/clientconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This doesn't actually check for the existence of a specific deployment, but any deployment
func checkDeploymentExists(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := clientconn.UnauthenticatedGRPCConnection(apiEndpoint)
	require.NoError(t, err)

	service := v1.NewDeploymentServiceClient(conn)

	// 10 seconds should be enough for a deployment to be returned
	for i := 0; i < 5; i++ {
		resp, err := service.GetDeployments(ctx, &v1.GetDeploymentsRequest{})
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

	out, err := os.Create("backup.db")
	require.NoError(t, err)
	defer os.Remove("backup.db")

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	resp, err := client.Get("https://" + apiEndpoint + "/db/backup")
	require.NoError(t, err)
	defer resp.Body.Close()
	_, err = io.Copy(out, resp.Body)
	require.NoError(t, err)

	out.Close()

	b, err := boltdb.New("backup.db")
	require.NoError(t, err)

	deployments, err := b.GetDeployments(&v1.GetDeploymentsRequest{})
	require.NoError(t, err)
	assert.NotEmpty(t, deployments)
}
