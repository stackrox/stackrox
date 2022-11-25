package metrics

import (
	"context"
	"runtime/debug"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	log = logging.LoggerForModule()
)

type grpcMetricsImpl struct {
	allMetricsMutex sync.Mutex
	allMetrics      map[string]*perPathGRPCMetrics
}

type perPathGRPCMetrics struct {
	normalInvocationStats map[codes.Code]int64

	panics *lru.Cache[string, int64]
}

func (g *grpcMetricsImpl) getOrCreateAllMetrics(path string) *perPathGRPCMetrics {
	perPathMetric := g.allMetrics[path]
	if perPathMetric == nil {
		panicLRU, err := lru.New[string, int64](cacheSize)
		err = utils.ShouldErr(errors.Wrap(err, "error creating an lru"))
		if err != nil {
			return nil
		}
		perPathMetric = &perPathGRPCMetrics{
			normalInvocationStats: make(map[codes.Code]int64),
			panics:                panicLRU,
		}
		g.allMetrics[path] = perPathMetric
	}

	return perPathMetric
}

func (g *grpcMetricsImpl) updateInternalMetric(path string, responseCode codes.Code) {
	g.allMetricsMutex.Lock()
	defer g.allMetricsMutex.Unlock()

	perPathMetric := g.getOrCreateAllMetrics(path)
	if perPathMetric == nil {
		return
	}
	perPathMetric.normalInvocationStats[responseCode]++
}

func anyToError(x interface{}) error {
	if x == nil {
		return errors.New("nil panic reason")
	}
	if err, ok := x.(error); ok {
		return err
	}
	return errors.Errorf("%v", x)
}

func (g *grpcMetricsImpl) convertPanicToError(p interface{}) error {
	err := anyToError(p)
	utils.Should(errors.Errorf("Caught panic in gRPC call. Reason: %v. Stack trace:\n%s", err, string(debug.Stack())))
	return err
}

func (g *grpcMetricsImpl) UnaryMonitoringInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	panicked := true
	defer func() {
		r := recover()
		if r == nil && !panicked {
			return
		}
		// Convert the panic to an error
		err = g.convertPanicToError(r)

		// Keep stats about the location and number of panics
		panicLocation := getPanicLocation(1)
		path := info.FullMethod
		g.allMetricsMutex.Lock()
		defer g.allMetricsMutex.Unlock()
		perPathMetric := g.getOrCreateAllMetrics(path)
		if perPathMetric == nil {
			panic(r)
		}
		apiPanic, ok := perPathMetric.panics.Get(panicLocation)
		if !ok {
			apiPanic = int64(0)
		}
		perPathMetric.panics.Add(panicLocation, apiPanic+1)
		panic(r)
	}()
	resp, err = handler(ctx, req)

	errStatus, _ := status.FromError(err)
	responseCode := errStatus.Code()
	g.updateInternalMetric(info.FullMethod, responseCode)

	panicked = false
	return
}

// GetMetrics returns copies of the internal metric maps
func (g *grpcMetricsImpl) GetMetrics() (map[string]map[codes.Code]int64, map[string]map[string]int64) {
	g.allMetricsMutex.Lock()
	defer g.allMetricsMutex.Unlock()
	externalMetrics := make(map[string]map[codes.Code]int64, len(g.allMetrics))
	externalPanics := make(map[string]map[string]int64, len(g.allMetrics))
	for path, perPathMetric := range g.allMetrics {
		externalCodeMap := make(map[codes.Code]int64, len(perPathMetric.normalInvocationStats))
		for responseCode, count := range perPathMetric.normalInvocationStats {
			externalCodeMap[responseCode] = count
		}
		if len(externalCodeMap) > 0 {
			externalMetrics[path] = externalCodeMap
		}

		panicLocations := perPathMetric.panics.Keys()
		panicList := make(map[string]int64, len(panicLocations))
		for _, panicLocation := range panicLocations {
			if panicCount, ok := perPathMetric.panics.Get(panicLocation); ok {
				panicList[panicLocation] = panicCount
			}
		}
		if len(panicList) > 0 {
			externalPanics[path] = panicList
		}
	}

	return externalMetrics, externalPanics
}
