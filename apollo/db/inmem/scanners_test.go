package inmem

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func testScanners(t *testing.T, insertStorage, retrievalStorage db.ScannerStorage) {
	scanners := []*v1.Scanner{
		{
			Name:     "scanner1",
			Endpoint: "https://endpoint1",
		},
		{
			Name:     "scanner2",
			Endpoint: "https://endpoint2",
		},
	}

	// Test Add
	for _, r := range scanners {
		id, err := insertStorage.AddScanner(r)
		assert.NoError(t, err)
		assert.NotEmpty(t, id)
	}

	for _, r := range scanners {
		got, exists, err := retrievalStorage.GetScanner(r.GetId())
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, got, r)
	}

	// Test Update
	for _, r := range scanners {
		r.Endpoint += "/api"
	}

	for _, r := range scanners {
		assert.NoError(t, insertStorage.UpdateScanner(r))
	}

	for _, r := range scanners {
		got, exists, err := retrievalStorage.GetScanner(r.GetId())
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, got, r)
	}

	// Test Remove
	for _, r := range scanners {
		assert.NoError(t, insertStorage.RemoveScanner(r.GetId()))
	}

	for _, r := range scanners {
		_, exists, err := retrievalStorage.GetScanner(r.GetId())
		assert.NoError(t, err)
		assert.False(t, exists)
	}
}

func TestScannersPersistence(t *testing.T) {
	t.Parallel()
	persistent, err := createBoltDB()
	if err != nil {
		t.Fatal(err)
	}
	storage := newScannerStore(persistent)
	testScanners(t, storage, persistent)
}

func TestScanners(t *testing.T) {
	t.Parallel()
	persistent, err := createBoltDB()
	if err != nil {
		t.Fatal(err)
	}
	storage := newScannerStore(persistent)
	testScanners(t, storage, storage)
}

func TestScannersFiltering(t *testing.T) {
	t.Parallel()
	persistent, err := createBoltDB()
	if err != nil {
		t.Fatal(err)
	}
	storage := newScannerStore(persistent)
	scanners := []*v1.Scanner{
		{
			Name:     "scanner1",
			Endpoint: "https://endpoint1",
		},
		{
			Name:     "scanner2",
			Endpoint: "https://endpoint2",
		},
	}

	// Test Add
	for _, r := range scanners {
		id, err := storage.AddScanner(r)
		assert.NoError(t, err)
		assert.NotEmpty(t, id)
	}

	actualScanners, err := storage.GetScanners(&v1.GetScannersRequest{})
	assert.NoError(t, err)
	assert.ElementsMatch(t, scanners, actualScanners)
}
