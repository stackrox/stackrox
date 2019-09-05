package reprocessor

import (
	"context"
	"time"

	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/deployment/mappings"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/throttle"
	"github.com/stackrox/rox/pkg/uuid"
	"golang.org/x/time/rate"
)

var (
	log = logging.LoggerForModule()

	dedupeNamespace = uuid.NewV4()

	once sync.Once
	loop Loop

	maxInjectionDelay = 500 * time.Millisecond

	getDeploymentContext = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Deployment)))
)

// Singleton returns the singleton reprocessor loop
func Singleton() Loop {
	once.Do(func() {
		loop = NewLoop(connection.ManagerSingleton(), deploymentDS.Singleton())
	})
	return loop
}

// Loop combines periodically (every hour) runs enrichment and detection.
//go:generate mockgen-wrapper
type Loop interface {
	Start()
	ShortCircuit()
	Stop()
}

// NewLoop returns a new instance of a Loop.
func NewLoop(connManager connection.Manager, deployments deploymentDS.DataStore) Loop {
	return newLoopWithDuration(connManager, deployments, 4*time.Hour, 30*time.Minute, 15*time.Second)
}

// newLoopWithDuration returns a loop that ticks at the given duration.
// It is NOT exported, since we don't want clients to control the duration; it only exists as a separate function
// to enable testing.
func newLoopWithDuration(connManager connection.Manager, deployments deploymentDS.DataStore, enrichAndDetectDuration, enrichAndDetectInjectionPeriod, deploymentRiskDuration time.Duration) Loop {
	return &loopImpl{
		enrichAndDetectTickerDuration:  enrichAndDetectDuration,
		enrichAndDetectInjectionPeriod: enrichAndDetectInjectionPeriod,

		deployments: deployments,

		stopChan:  concurrency.NewSignal(),
		stopped:   concurrency.NewSignal(),
		shortChan: make(chan struct{}),

		connManager: connManager,
		throttler:   throttle.NewDropThrottle(time.Second),
	}
}

type loopImpl struct {
	enrichAndDetectTickerDuration  time.Duration
	enrichAndDetectInjectionPeriod time.Duration
	enrichAndDetectTicker          *time.Ticker

	deployments deploymentDS.DataStore

	shortChan chan struct{}
	stopChan  concurrency.Signal
	stopped   concurrency.Signal

	connManager connection.Manager
	throttler   throttle.DropThrottle
}

// Start starts the enrich and detect loop.
func (l *loopImpl) Start() {
	l.enrichAndDetectTicker = time.NewTicker(l.enrichAndDetectTickerDuration)
	go l.loop()
}

// Stop stops the enrich and detect loop.
func (l *loopImpl) Stop() {
	l.stopChan.Signal()
	l.stopped.Wait()
}

func (l *loopImpl) ShortCircuit() {
	select {
	case l.shortChan <- struct{}{}:
	case <-l.stopped.Done():
	}
}

func (l *loopImpl) sendDeployments(injectionPeriod time.Duration, deploymentIDs ...string) {
	query := search.NewQueryBuilder().AddStringsHighlighted(search.ClusterID, search.WildcardString)
	if len(deploymentIDs) > 0 {
		query = query.AddDocIDs(deploymentIDs...)
	}

	results, err := l.deployments.SearchDeployments(getDeploymentContext, query.ProtoQuery())
	if err != nil {
		log.Errorf("error getting results for reprocessing: %v", err)
		return
	}

	path, ok := mappings.OptionsMap.Get(search.ClusterID.String())
	if !ok {
		panic("No Cluster ID option for deployments")
	}

	var injectionLimiter *rate.Limiter
	if injectionPeriod != 0 && len(results) != 0 {
		calculatedRate := time.Duration(l.enrichAndDetectInjectionPeriod.Nanoseconds() / int64(len(results)))
		if calculatedRate > maxInjectionDelay {
			calculatedRate = maxInjectionDelay
		}
		injectionLimiter = rate.NewLimiter(rate.Every(calculatedRate), 1)
	}

	for _, r := range results {
		clusterIDs := r.FieldToMatches[path.FieldPath].GetValues()
		if len(clusterIDs) == 0 {
			log.Error("no cluster id found in fields")
			continue
		}

		conn := l.connManager.GetConnection(clusterIDs[0])
		if conn == nil {
			continue
		}

		dedupeKey := uuid.NewV5(dedupeNamespace, r.Id).String()
		if injectionLimiter != nil {
			_ = injectionLimiter.Wait(context.Background())
		}

		msg := &central.MsgFromSensor{
			HashKey:   r.Id,
			DedupeKey: dedupeKey,
			Msg: &central.MsgFromSensor_Event{
				Event: &central.SensorEvent{
					Resource: &central.SensorEvent_ReprocessDeployment{
						ReprocessDeployment: &central.ReprocessDeployment{
							DeploymentId: r.Id,
						},
					},
				},
			},
		}

		conn.InjectMessageIntoQueue(msg)
	}
}

func (l *loopImpl) loop() {
	defer l.stopped.Signal()
	defer l.enrichAndDetectTicker.Stop()
	for {
		select {
		case <-l.stopChan.Done():
			return
		case <-l.shortChan:
			l.sendDeployments(0)
		case <-l.enrichAndDetectTicker.C:
			l.sendDeployments(l.enrichAndDetectInjectionPeriod)
		}
	}
}
