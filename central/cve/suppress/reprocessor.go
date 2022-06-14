package suppress

import (
	"context"
	"fmt"
	"time"

	legacyImageCVEDataStore "github.com/stackrox/rox/central/cve/datastore"
	cveDataStore "github.com/stackrox/rox/central/cve/image/datastore"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()

	once sync.Once
	loop CVEUnsuppressLoop

	// This cannot be tested without building complete graph, hence this elevated context
	reprocessorCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
)

// CVEUnsuppressLoop periodically runs cve unsuppression
type CVEUnsuppressLoop interface {
	Start()
	Stop()
}

// Singleton returns the singleton reprocessor loop
func Singleton() CVEUnsuppressLoop {
	var imageCVEDataStore cveDataStore.DataStore
	if features.PostgresDatastore.Enabled() {
		imageCVEDataStore = cveDataStore.Singleton()
	} else {
		imageCVEDataStore = legacyImageCVEDataStore.Singleton()
	}

	once.Do(func() {
		// TODO: Attach other CVE stores.
		loop = NewLoop(imageCVEDataStore)
	})
	return loop
}

// NewLoop returns a new instance of a Loop.
func NewLoop(cves cveDataStore.DataStore) CVEUnsuppressLoop {
	// ticker duration is set to 1 hour since the smallest time unit for suppress expiry is 1 day.
	return newLoopWithDuration(cves, time.Hour)
}

// newLoopWithDuration returns a loop that ticks at the given duration.
// It is NOT exported, since we don't want clients to control the duration; it only exists as a separate function
// to enable testing.
func newLoopWithDuration(cves cveDataStore.DataStore, tickerDuration time.Duration) CVEUnsuppressLoop {
	return &cveUnsuppressLoopImpl{
		cves:                      cves,
		cveSuppressTickerDuration: tickerDuration,

		stopChan: concurrency.NewSignal(),
		stopped:  concurrency.NewSignal(),
	}
}

type cveUnsuppressLoopImpl struct {
	cveSuppressTickerDuration time.Duration
	cveSuppressTicker         *time.Ticker

	cves cveDataStore.DataStore

	stopChan concurrency.Signal
	stopped  concurrency.Signal
}

// Start starts the CVE unsuppress loop.
func (l *cveUnsuppressLoopImpl) Start() {
	l.cveSuppressTicker = time.NewTicker(l.cveSuppressTickerDuration)
	go l.loop()
}

// Stop stops the CVE unsuppress loop.
func (l *cveUnsuppressLoopImpl) Stop() {
	l.stopChan.Signal()
	l.stopped.Wait()
}

func (l *cveUnsuppressLoopImpl) unsuppressCVEsWithExpiredSuppressState() {
	if l.stopped.IsDone() {
		return
	}

	cves, err := l.getCVEsWithExpiredSuppressState()
	if err != nil {
		log.Errorf("error retrieving CVEs for reprocessing: %v", err)
		return
	}
	if len(cves) == 0 {
		return
	}

	if err := l.cves.Unsuppress(reprocessorCtx, cves...); err != nil {
		log.Errorf("error unsuppressing CVEs %+s: %v", cves, err)
		return
	}
	log.Infof("Successfully unsuppressed %d CVEs", len(cves))
}

func (l *cveUnsuppressLoopImpl) getCVEsWithExpiredSuppressState() ([]string, error) {
	// TODO: ROX-4072: change the format to 01/02/2006 15:04:05 MST once timestamp query is supported
	now := fmt.Sprintf("<%s", time.Now().Format("01/02/2006 MST"))
	q := search.NewQueryBuilder().AddGenericTypeLinkedFields(
		[]search.FieldLabel{search.CVESuppressed, search.CVESuppressExpiry}, []interface{}{true, now}).ProtoQuery()
	results, err := l.cves.Search(reprocessorCtx, q)

	if err != nil || len(results) == 0 {
		return nil, err
	}
	return search.ResultsToIDs(results), nil
}

func (l *cveUnsuppressLoopImpl) loop() {
	defer l.stopped.Signal()
	defer l.cveSuppressTicker.Stop()

	go l.unsuppressCVEsWithExpiredSuppressState()
	for {
		select {
		case <-l.stopChan.Done():
			return
		case <-l.cveSuppressTicker.C:
			l.unsuppressCVEsWithExpiredSuppressState()
		}
	}
}
