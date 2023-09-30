package injector

import (
	"testing"
	"time"

	mockstore "github.com/stackrox/rox/central/administration/usage/datastore/securedunits/mocks"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestInjectorStop(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	store := mockstore.NewMockDataStore(mockCtrl)

	var done bool
	ticker := make(chan time.Time)
	defer close(ticker)

	i := &injectorImpl{
		tickChan:       ticker,
		onStop:         func() { done = true },
		ds:             store,
		stop:           concurrency.NewSignal(),
		gatherersGroup: &sync.WaitGroup{},
	}

	i.Start()
	assert.False(t, done)

	store.EXPECT().AggregateAndReset(gomock.Any())
	store.EXPECT().Add(gomock.Any(), gomock.Any())
	ticker <- time.Now()
	assert.False(t, done)

	store.EXPECT().AggregateAndReset(gomock.Any())
	store.EXPECT().Add(gomock.Any(), gomock.Any())
	ticker <- time.Now()
	assert.False(t, done)

	i.Stop()
	assert.True(t, done)
}
