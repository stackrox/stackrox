package injector

import (
	"context"
	"testing"
	"time"

	mockstore "github.com/stackrox/rox/central/administration/usage/datastore/securedunits/mocks"
	"github.com/stackrox/rox/pkg/backgroundworker"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestInjectorGather(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	store := mockstore.NewMockDataStore(mockCtrl)

	i := &injectorImpl{ds: store}

	store.EXPECT().AggregateAndReset(gomock.Any())
	store.EXPECT().Add(gomock.Any(), gomock.Any())
	assert.NoError(t, i.gather(context.Background()))
}

func TestInjectorGatherPropagatesAggregateError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	store := mockstore.NewMockDataStore(mockCtrl)

	i := &injectorImpl{ds: store}

	store.EXPECT().AggregateAndReset(gomock.Any()).Return(nil, assert.AnError)
	assert.Error(t, i.gather(context.Background()))
}

func TestInjectorStartStop(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	store := mockstore.NewMockDataStore(mockCtrl)

	store.EXPECT().AggregateAndReset(gomock.Any()).MinTimes(1)
	store.EXPECT().Add(gomock.Any(), gomock.Any()).MinTimes(1)

	impl := &injectorImpl{ds: store}
	impl.worker = &backgroundworker.PeriodicWorker{
		Name:     "test-usage-metrics-injector",
		Interval: 10 * time.Millisecond,
		Run:      impl.gather,
	}

	impl.Start()
	time.Sleep(35 * time.Millisecond)
	impl.Stop()
}
