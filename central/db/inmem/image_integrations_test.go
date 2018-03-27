package inmem

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func testIntegrations(t *testing.T, insertStorage, retrievalStorage db.ImageIntegrationStorage) {
	integrations := []*v1.ImageIntegration{
		{
			Name: "registry1",
			Config: map[string]string{
				"endpoint": "https://endpoint1",
			},
		},
		{
			Name: "registry2",
			Config: map[string]string{
				"endpoint": "https://endpoint2",
			},
		},
	}

	// Test Add
	for _, r := range integrations {
		id, err := insertStorage.AddImageIntegration(r)
		assert.NoError(t, err)
		assert.NotEmpty(t, id)
	}
	for _, r := range integrations {
		got, exists, err := retrievalStorage.GetImageIntegration(r.GetId())
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, got, r)
	}

	// Test Update
	for _, r := range integrations {
		r.Name += "/api"
	}

	for _, r := range integrations {
		assert.NoError(t, insertStorage.UpdateImageIntegration(r))
	}

	for _, r := range integrations {
		got, exists, err := retrievalStorage.GetImageIntegration(r.GetId())
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, got, r)
	}

	// Test Remove
	for _, r := range integrations {
		assert.NoError(t, insertStorage.RemoveImageIntegration(r.GetId()))
	}

	for _, r := range integrations {
		_, exists, err := retrievalStorage.GetImageIntegration(r.GetId())
		assert.NoError(t, err)
		assert.False(t, exists)
	}
}

func TestIntegrationsPersistence(t *testing.T) {
	t.Parallel()
	persistent, err := createBoltDB()
	if err != nil {
		t.Fatal(err)
	}
	storage := newImageIntegrationStore(persistent)
	testIntegrations(t, storage, persistent)
}

func TestIntegrations(t *testing.T) {
	t.Parallel()
	persistent, err := createBoltDB()
	if err != nil {
		t.Fatal(err)
	}
	storage := newImageIntegrationStore(persistent)
	testIntegrations(t, storage, storage)
}

func TestIntegrationsFiltering(t *testing.T) {
	t.Parallel()
	persistent, err := createBoltDB()
	if err != nil {
		t.Fatal(err)
	}
	storage := newImageIntegrationStore(persistent)
	integrations := []*v1.ImageIntegration{
		{
			Name: "registry1",
			Config: map[string]string{
				"endpoint": "https://endpoint1",
			},
		},
		{
			Name: "registry2",
			Config: map[string]string{
				"endpoint": "https://endpoint2",
			},
		},
	}

	// Test Add
	for _, r := range integrations {
		id, err := storage.AddImageIntegration(r)
		assert.NoError(t, err)
		assert.NotEmpty(t, id)
	}

	actualIntegrations, err := storage.GetImageIntegrations(&v1.GetImageIntegrationsRequest{})
	assert.NoError(t, err)
	assert.ElementsMatch(t, integrations, actualIntegrations)

}
