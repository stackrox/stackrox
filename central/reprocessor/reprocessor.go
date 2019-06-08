package reprocessor

import (
	"time"

	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/throttle"
)

var (
	log = logging.LoggerForModule()

	once sync.Once
	loop Loop
)

// Singleton returns the singleton reprocessor loop
func Singleton() Loop {
	once.Do(func() {
		loop = NewLoop(connection.ManagerSingleton())
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
func NewLoop(connManager connection.Manager) Loop {
	return newLoopWithDuration(connManager, time.Hour, 15*time.Second)
}

// newLoopWithDuration returns a loop that ticks at the given duration.
// It is NOT exported, since we don't want clients to control the duration; it only exists as a separate function
// to enable testing.
func newLoopWithDuration(connManager connection.Manager, enrichAndDetectDuration, deploymentRiskDuration time.Duration) Loop {
	return &loopImpl{
		enrichAndDetectTickerDuration: enrichAndDetectDuration,
		deploymenRiskTickerDuration:   deploymentRiskDuration,

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
	l.throttler.Run(func() { l.sendRisk() })
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

func (l *loopImpl) sendEnrichAndDetect(deploymentIDs ...string) {
	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_ReprocessDeployments{
			ReprocessDeployments: &central.ReprocessDeployments{
				DeploymentIds: deploymentIDs,
				Target: &central.ReprocessDeployments_All{
					All: &central.ReprocessDeployments_AllTarget{},
				},
			},
		},
	}
	l.sendMessageToPipeline(msg)
}

func (l *loopImpl) sendRisk(deploymentIDs ...string) {
	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_ReprocessDeployments{
			ReprocessDeployments: &central.ReprocessDeployments{
				DeploymentIds: deploymentIDs,
				Target: &central.ReprocessDeployments_Risk{
					Risk: &central.ReprocessDeployments_RiskTarget{},
				},
			},
		},
	}
	l.sendMessageToPipeline(msg)
}

func (l *loopImpl) sendMessageToPipeline(msg *central.MsgFromSensor) {
	for _, conn := range l.connManager.GetActiveConnections() {
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
			l.sendEnrichAndDetect()
		case <-l.enrichAndDetectTicker.C:
			l.sendEnrichAndDetect()
		case <-l.deploymentRiskTicker.C:
			l.deploymentRiskLock.Lock()
			if l.deploymentRiskSet.Cardinality() > 0 {
				l.sendRisk(l.deploymentRiskSet.AsSlice()...)
				l.deploymentRiskSet.Clear()
			}
			l.deploymentRiskLock.Unlock()
		}
	}
}
