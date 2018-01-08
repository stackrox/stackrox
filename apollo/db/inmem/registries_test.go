package inmem

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func testRegistries(t *testing.T, insertStorage, retrievalStorage db.RegistryStorage) {
	registries := []*v1.Registry{
		{
			Name:     "registry1",
			Endpoint: "https://endpoint1",
		},
		{
			Name:     "registry2",
			Endpoint: "https://endpoint2",
		},
	}

	// Test Add
	for _, r := range registries {
		assert.NoError(t, insertStorage.AddRegistry(r))
	}
	// Verify insertion multiple times does not deadlock and causes an error
	for _, r := range registries {
		assert.Error(t, insertStorage.AddRegistry(r))
	}
	for _, r := range registries {
		got, exists, err := retrievalStorage.GetRegistry(r.Name)
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, got, r)
	}

	// Test Update
	for _, r := range registries {
		r.Endpoint += "/api"
	}

	for _, r := range registries {
		assert.NoError(t, insertStorage.UpdateRegistry(r))
	}

	for _, r := range registries {
		got, exists, err := retrievalStorage.GetRegistry(r.Name)
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, got, r)
	}

	// Test Remove
	for _, r := range registries {
		assert.NoError(t, insertStorage.RemoveRegistry(r.Name))
	}

	for _, r := range registries {
		_, exists, err := retrievalStorage.GetRegistry(r.Name)
		assert.NoError(t, err)
		assert.False(t, exists)
	}
}

func TestRegistriesPersistence(t *testing.T) {
	t.Parallel()
	persistent, err := createBoltDB()
	if err != nil {
		t.Fatal(err)
	}
	storage := newRegistryStore(persistent)
	testRegistries(t, storage, persistent)
}

func TestRegistries(t *testing.T) {
	t.Parallel()
	persistent, err := createBoltDB()
	if err != nil {
		t.Fatal(err)
	}
	storage := newRegistryStore(persistent)
	testRegistries(t, storage, storage)
}

func TestRegistriesFiltering(t *testing.T) {
	t.Parallel()
	persistent, err := createBoltDB()
	if err != nil {
		t.Fatal(err)
	}
	storage := newRegistryStore(persistent)
	registries := []*v1.Registry{
		{
			Name:     "registry1",
			Endpoint: "https://endpoint1",
		},
		{
			Name:     "registry2",
			Endpoint: "https://endpoint2",
		},
	}

	// Test Add
	for _, r := range registries {
		assert.NoError(t, storage.AddRegistry(r))
	}

	actualRegistries, err := storage.GetRegistries(&v1.GetRegistriesRequest{})
	assert.NoError(t, err)
	assert.Equal(t, registries, actualRegistries)

}
