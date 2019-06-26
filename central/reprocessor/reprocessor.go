package reprocessor

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/deployment/mappings"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/throttle"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	log = logging.LoggerForModule()

	riskDedupeNamespace = uuid.NewV4()
	dedupeNamespace     = uuid.NewV4()

	once sync.Once
	loop Loop

	getDeploymentsContext = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Deployment)))
)

// Singleton returns the singleton reprocessor loop
func Singleton() Loop {
	once.Do(func() {
		loop = NewLoop(connection.ManagerSingleton(), datastore.Singleton())
	})
	return loop
}

// Loop combines periodically (every hour) runs enrichment and detection.
//go:generate mockgen-wrapper Loop
type Loop interface {
	Start()
	ShortCircuit()
	Stop()

	ReprocessRisk()
	ReprocessRiskForDeployments(deploymentIDs ...string)
}

// NewLoop returns a new instance of a Loop.
func NewLoop(connManager connection.Manager, deployments datastore.DataStore) Loop {
	return newLoopWithDuration(connManager, deployments, time.Hour, 15*time.Second)
}

// newLoopWithDuration returns a loop that ticks at the given duration.
// It is NOT exported, since we don't want clients to control the duration; it only exists as a separate function
// to enable testing.
func newLoopWithDuration(connManager connection.Manager, deployments datastore.DataStore, enrichAndDetectDuration, deploymentRiskDuration time.Duration) Loop {
	return &loopImpl{
		enrichAndDetectTickerDuration: enrichAndDetectDuration,
		deploymenRiskTickerDuration:   deploymentRiskDuration,

		deployments:       deployments,
		deploymentRiskSet: set.NewStringSet(),

		stopChan:  concurrency.NewSignal(),
		stopped:   concurrency.NewSignal(),
		shortChan: make(chan struct{}),

		connManager: connManager,
		throttler:   throttle.NewDropThrottle(time.Second),
	}
}

type loopImpl struct {
	enrichAndDetectTickerDuration time.Duration
	enrichAndDetectTicker         *time.Ticker

	deployments                 datastore.DataStore
	deploymentRiskSet           set.StringSet
	deploymentRiskLock          sync.Mutex
	deploymentRiskTicker        *time.Ticker
	deploymenRiskTickerDuration time.Duration

	shortChan chan struct{}
	stopChan  concurrency.Signal
	stopped   concurrency.Signal

	connManager connection.Manager
	throttler   throttle.DropThrottle
}

func (l *loopImpl) ReprocessRisk() {
	l.throttler.Run(func() { l.sendDeployments(true) })
}

func (l *loopImpl) ReprocessRiskForDeployments(deploymentIDs ...string) {
	l.deploymentRiskLock.Lock()
	defer l.deploymentRiskLock.Unlock()
	l.deploymentRiskSet.AddAll(deploymentIDs...)
}

// Start starts the enrich and detect loop.
func (l *loopImpl) Start() {
	l.enrichAndDetectTicker = time.NewTicker(l.enrichAndDetectTickerDuration)
	l.deploymentRiskTicker = time.NewTicker(l.deploymenRiskTickerDuration)
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

func (l *loopImpl) sendDeployments(riskOnly bool, deploymentIDs ...string) {
	query := search.NewQueryBuilder().AddStringsHighlighted(search.ClusterID, search.WildcardString)
	if len(deploymentIDs) > 0 {
		query = query.AddDocIDs(deploymentIDs...)
	}

	results, err := l.deployments.SearchDeployments(getDeploymentsContext, query.ProtoQuery())
	if err != nil {
		log.Errorf("error getting results for reprocessing: %v", err)
		return
	}

	path, ok := mappings.OptionsMap.Get(search.ClusterID.String())
	if !ok {
		panic("No Cluster ID option for deployments")
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

		var dedupeKey string
		if riskOnly {
			dedupeKey = uuid.NewV5(riskDedupeNamespace, r.Id).String()
		} else {
			dedupeKey = uuid.NewV5(dedupeNamespace, r.Id).String()
		}

		msg := &central.MsgFromSensor{
			HashKey:   r.Id,
			DedupeKey: dedupeKey,
			Msg: &central.MsgFromSensor_Event{
				Event: &central.SensorEvent{
					Resource: &central.SensorEvent_ReprocessDeployment{
						ReprocessDeployment: &central.ReprocessDeployment{
							DeploymentId: r.Id,
							RiskOnly:     riskOnly,
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
	defer l.deploymentRiskTicker.Stop()
	for {
		select {
		case <-l.stopChan.Done():
			return
		case <-l.shortChan:
			l.sendDeployments(false)
		case <-l.enrichAndDetectTicker.C:
			l.sendDeployments(false)
		case <-l.deploymentRiskTicker.C:
			l.deploymentRiskLock.Lock()
			if l.deploymentRiskSet.Cardinality() > 0 {
				l.sendDeployments(true, l.deploymentRiskSet.AsSlice()...)
				l.deploymentRiskSet.Clear()
			}
			l.deploymentRiskLock.Unlock()
		}
	}
}
