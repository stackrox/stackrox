package gatherers

import (
	"github.com/stackrox/rox/pkg/grpc/metrics"
	"github.com/stackrox/rox/pkg/telemetry/data"
)

type grpcGatherer struct {
	grpcMetrics metrics.GRPCMetrics
}

func newGRPCGatherer(grpcMetrics metrics.GRPCMetrics) *grpcGatherer {
	return &grpcGatherer{
		grpcMetrics: grpcMetrics,
	}
}

func (g *grpcGatherer) Gather() []*data.GRPCMethod {
	statMap, panicMap := g.grpcMetrics.GetMetrics()
	apiStats := make(map[string]*data.GRPCMethod, len(statMap))
	for name, statusMap := range statMap {
		stat := &data.GRPCMethod{
			Method:            name,
			NormalInvocations: make([]*data.GRPCInvocationStats, 0, len(statusMap)),
		}
		for status, count := range statusMap {
			stat.NormalInvocations = append(stat.NormalInvocations, &data.GRPCInvocationStats{
				Code:  status,
				Count: count,
			})
		}
		apiStats[name] = stat
	}

	for name, panicLocationMap := range panicMap {
		stat, ok := apiStats[name]
		if !ok {
			stat = &data.GRPCMethod{
				Method:           name,
				PanicInvocations: make([]*data.PanicStats, 0, len(panicLocationMap)),
			}
			apiStats[name] = stat
		}
		stat.PanicInvocations = makePanics(panicLocationMap)
	}

	grpcList := make([]*data.GRPCMethod, 0, len(apiStats))
	for _, grpcStat := range apiStats {
		grpcList = append(grpcList, grpcStat)
	}
	return grpcList
}
