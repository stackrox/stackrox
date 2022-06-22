package suppress

import (
	"context"
	"fmt"
	"time"

	legacyImageCVEDataStore "github.com/stackrox/rox/central/cve/datastore"
	imageCVEDataStore "github.com/stackrox/rox/central/cve/image/datastore"
	nodeCVEDataStore "github.com/stackrox/rox/central/cve/node/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/cve"
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

type vulnsStore interface {
	Unsuppress(ctx context.Context, ids ...string) error
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
}

// Singleton returns the singleton reprocessor loop
func Singleton() CVEUnsuppressLoop {

	once.Do(func() {
		if features.PostgresDatastore.Enabled() {
			// TODO: Attach cluster CVE store.
			loop = NewLoop(imageCVEDataStore.Singleton(), nodeCVEDataStore.Singleton())
		} else {
			loop = NewLoop(legacyImageCVEDataStore.Singleton())
		}
	})
	return loop
}

// NewLoop returns a new instance of a Loop.
func NewLoop(cveStores ...vulnsStore) CVEUnsuppressLoop {
	// ticker duration is set to 1 hour since the smallest time unit for suppress expiry is 1 day.
	return newLoopWithDuration(time.Hour, cveStores...)
}

// newLoopWithDuration returns a loop that ticks at the given duration.
// It is NOT exported, since we don't want clients to control the duration; it only exists as a separate function
// to enable testing.
func newLoopWithDuration(tickerDuration time.Duration, cveStores ...vulnsStore) CVEUnsuppressLoop {
	return &cveUnsuppressLoopImpl{
		cveStores:                 cveStores,
		cveSuppressTickerDuration: tickerDuration,

		stopChan: concurrency.NewSignal(),
		stopped:  concurrency.NewSignal(),
	}
}

type cveUnsuppressLoopImpl struct {
	cveSuppressTickerDuration time.Duration
	cveSuppressTicker         *time.Ticker

	cveStores []vulnsStore

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

	totalUnsuppressedCVEs := 0
	for _, cveStore := range l.cveStores {
		cves, err := getCVEsWithExpiredSuppressState(cveStore)
		if err != nil {
			log.Errorf("error retrieving CVEs for reprocessing: %v", err)
			continue
		}
		if len(cves) == 0 {
			continue
		}

		if err := cveStore.Unsuppress(reprocessorCtx, cves...); err != nil {
			log.Errorf("error unsuppressing CVEs %+s: %v", cves, err)
			continue
		}
		totalUnsuppressedCVEs += len(cves)
	}
	log.Infof("Successfully unsuppressed %d CVEs", totalUnsuppressedCVEs)
}

func getCVEsWithExpiredSuppressState(cveStore vulnsStore) ([]string, error) {
	// TODO: ROX-4072: change the format to 01/02/2006 15:04:05 MST once timestamp query is supported
	now := fmt.Sprintf("<%s", time.Now().Format("01/02/2006 MST"))
	q := search.NewQueryBuilder().AddGenericTypeLinkedFields(
		[]search.FieldLabel{search.CVESuppressed, search.CVESuppressExpiry}, []interface{}{true, now}).ProtoQuery()
	results, err := cveStore.Search(reprocessorCtx, q)
	if err != nil || len(results) == 0 {
		return nil, err
	}

	if features.PostgresDatastore.Enabled() {
		cves := make([]string, 0, len(results))
		for _, res := range results {
			cve, _ := cve.IDToParts(res.ID)
			cves = append(cves, cve)
		}
		return cves, nil
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
