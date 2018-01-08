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
		assert.NoError(t, insertStorage.AddScanner(r))
	}
	// Verify insertion multiple times does not deadlock and causes an error
	for _, r := range scanners {
		assert.Error(t, insertStorage.AddScanner(r))
	}

	for _, r := range scanners {
		got, exists, err := retrievalStorage.GetScanner(r.Name)
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
		got, exists, err := retrievalStorage.GetScanner(r.Name)
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, got, r)
	}

	// Test Remove
	for _, r := range scanners {
		assert.NoError(t, insertStorage.RemoveScanner(r.Name))
	}

	for _, r := range scanners {
		_, exists, err := retrievalStorage.GetScanner(r.Name)
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
		assert.NoError(t, storage.AddScanner(r))
	}

	actualScanners, err := storage.GetScanners(&v1.GetScannersRequest{})
	assert.NoError(t, err)
	assert.Equal(t, scanners, actualScanners)
}
