package injector

import (
	"testing"
	"time"

	mockstore "github.com/stackrox/rox/central/administration/usage/datastore/securedunits/mocks"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestInjectorStop(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	store := mockstore.NewMockDataStore(mockCtrl)

	i := &injectorImpl{
		ds:             store,
		gatherersGroup: &sync.WaitGroup{},
	}

	i.Start()

	// The worker is now running in the background.
	// We can trigger it manually or just verify that Stop() completes.
	// Since we don't have a good way to trigger the worker without waiting for the interval,
	// we'll just verify that Start/Stop complete without error.
	assert.NotNil(t, i.worker)

	// Give the worker a moment to start.
	time.Sleep(10 * time.Millisecond)

	// Stop should complete gracefully.
	i.Stop()
	assert.Nil(t, i.worker.Stopped().Err())
}
