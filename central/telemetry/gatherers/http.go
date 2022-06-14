package gatherers

import (
	"github.com/stackrox/rox/pkg/grpc/metrics"
	"github.com/stackrox/rox/pkg/telemetry/data"
)

type httpGatherer struct {
	httpMetrics metrics.HTTPMetrics
}

func newHTTPGatherer(httpMetrics metrics.HTTPMetrics) *httpGatherer {
	return &httpGatherer{
		httpMetrics: httpMetrics,
	}
}

func (h *httpGatherer) Gather() []*data.HTTPRoute {
	statMap, panicMap := h.httpMetrics.GetMetrics()
	apiStats := make(map[string]*data.HTTPRoute, len(statMap))
	for name, statusMap := range statMap {
		stat := &data.HTTPRoute{
			Route:             name,
			NormalInvocations: make([]*data.HTTPInvocationStats, 0, len(statusMap)),
		}
		for status, count := range statusMap {
			stat.NormalInvocations = append(stat.NormalInvocations, &data.HTTPInvocationStats{
				StatusCode: status,
				Count:      count,
			})
		}
		apiStats[name] = stat
	}

	for name, panicLocationMap := range panicMap {
		stat, ok := apiStats[name]
		if !ok {
			stat = &data.HTTPRoute{
				Route:            name,
				PanicInvocations: make([]*data.PanicStats, 0, len(panicLocationMap)),
			}
			apiStats[name] = stat
		}
		stat.PanicInvocations = makePanics(panicLocationMap)
	}

	httpList := make([]*data.HTTPRoute, 0, len(apiStats))
	for _, httpStat := range apiStats {
		httpList = append(httpList, httpStat)
	}
	return httpList
}
