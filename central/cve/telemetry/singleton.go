package telemetry

import (
	"context"
	"sync"
	"time"

	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/metrics"
)

var once sync.Once

var instance *trackImpl

func Singleton() *trackImpl {
	once.Do(func() {
		instance = &trackImpl{
			ds:        deploymentDS.Singleton(),
			cvssGauge: metrics.SetImageVulnCVSS,
		}
	})
	return instance
}

type trackImpl struct {
	ds         deploymentDS.DataStore
	stopSignal chan bool

	cvssGauge func(map[string]string, float64)
}

func (h *trackImpl) Start() {
	go h.track()
}

func (h *trackImpl) Stop() {
	close(h.stopSignal)
}

func (h *trackImpl) track() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for {
		select {
		case <-ticker.C:
			h.trackCvssMetrics(ctx)
		case <-h.stopSignal:
			return
		}
	}
}
