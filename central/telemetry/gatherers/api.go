package gatherers

import (
	"github.com/stackrox/rox/pkg/grpc/metrics"
	"github.com/stackrox/rox/pkg/telemetry/data"
)

type apiGatherer struct {
	grpcGatherer *grpcGatherer
	httpGatherer *httpGatherer
}

func newAPIGatherer(grpcMetrics metrics.GRPCMetrics, httpMetrics metrics.HTTPMetrics) *apiGatherer {
	return &apiGatherer{
		grpcGatherer: newGRPCGatherer(grpcMetrics),
		httpGatherer: newHTTPGatherer(httpMetrics),
	}
}

func (a *apiGatherer) Gather() *data.APIStats {
	return &data.APIStats{
		GRPC: a.grpcGatherer.Gather(),
		HTTP: a.httpGatherer.Gather(),
	}
}

func makePanics(panicMetricMap map[string]int64) []*data.PanicStats {
	panicList := make([]*data.PanicStats, 0, len(panicMetricMap))
	for location, count := range panicMetricMap {
		panicList = append(panicList, &data.PanicStats{
			PanicDesc: location,
			Count:     count,
		})
	}
	return panicList
}
