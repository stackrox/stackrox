package pruning

import (
	"testing"
	"time"

	configDS "github.com/stackrox/stackrox/central/config/datastore"
	"github.com/stackrox/stackrox/central/globalindex"
	"github.com/stackrox/stackrox/central/vulnerabilityrequest/cache"
	vulnReqDataStore "github.com/stackrox/stackrox/central/vulnerabilityrequest/datastore"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpiredVulnReqsPruning(t *testing.T) {
	db := rocksdbtest.RocksDBForT(t)
	defer rocksdbtest.TearDownRocksDB(db)

	bleveIndex, err := globalindex.MemOnlyIndex()
	require.NoError(t, err)

	datastore, err := vulnReqDataStore.NewForTestOnly(t, db, bleveIndex, cache.PendingReqsCacheSingleton(), cache.ActiveReqsCacheSingleton())
	require.NoError(t, err)

	oneMonthDayPastRetention := (30 + configDS.DefaultExpiredVulnReqRetention) * 24 * time.Hour
	oneDayPastRetention := (2 + configDS.DefaultExpiredVulnReqRetention) * 24 * time.Hour

	cases := []struct {
		name               string
		vulnRequests       []*storage.VulnerabilityRequest
		expectedDeletions  []string
		expectedRetentions []string
	}{
		{
			name: "not expired and fresh",
			vulnRequests: []*storage.VulnerabilityRequest{
				newVulnReq("req1", time.Minute, false),
				newVulnReq("req2", time.Minute, false),
			},
			expectedDeletions:  []string{},
			expectedRetentions: []string{"req1", "req2"},
		},
		{
			name: "not expired but older than retention period",
			vulnRequests: []*storage.VulnerabilityRequest{
				newVulnReq("req1", oneDayPastRetention, false),
				newVulnReq("req2", oneDayPastRetention, false),
			},
			expectedDeletions:  []string{},
			expectedRetentions: []string{"req1", "req2"},
		},
		{
			name: "expired and past retention period",
			vulnRequests: []*storage.VulnerabilityRequest{
				newVulnReq("req1", oneDayPastRetention, true),
				newVulnReq("req2", oneDayPastRetention, true),
			},
			expectedDeletions: []string{"req1", "req2"},
		},
		{
			name: "expired but not past retention period",
			vulnRequests: []*storage.VulnerabilityRequest{
				newVulnReq("req1", time.Minute, true),
				newVulnReq("req2", time.Minute, true),
			},
			expectedDeletions:  []string{},
			expectedRetentions: []string{"req1", "req2"},
		},
		{
			name: "expired; some past retention and some not past retention period",
			vulnRequests: []*storage.VulnerabilityRequest{
				newVulnReq("req1", time.Minute, true),
				newVulnReq("req2", oneDayPastRetention, true),
			},
			expectedDeletions:  []string{"req2"},
			expectedRetentions: []string{"req1"},
		},
		{
			name: "some expired and some not expired",
			vulnRequests: []*storage.VulnerabilityRequest{
				newVulnReq("req1", time.Minute, false),
				newVulnReq("req2", oneDayPastRetention, true),
				newVulnReq("req3", oneMonthDayPastRetention, true),
			},
			expectedDeletions:  []string{"req2", "req3"},
			expectedRetentions: []string{"req1"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, req := range c.vulnRequests {
				require.NoError(t, datastore.AddRequest(pruningCtx, req))
			}

			gci := &garbageCollectorImpl{
				vulnReqs: datastore,
			}
			gci.removeExpiredVulnRequests()

			if len(c.expectedDeletions) > 0 {
				results, err := datastore.Search(pruningCtx, search.NewQueryBuilder().AddDocIDs(c.expectedDeletions...).ProtoQuery())
				require.NoError(t, err)
				assert.Len(t, results, 0)
			}
			if len(c.expectedRetentions) > 0 {
				results, err := datastore.Search(pruningCtx, search.NewQueryBuilder().AddDocIDs(c.expectedRetentions...).ProtoQuery())
				require.NoError(t, err)
				assert.Len(t, results, len(c.expectedRetentions))
			}
		})
	}
}

func newVulnReq(id string, age time.Duration, expired bool) *storage.VulnerabilityRequest {
	return &storage.VulnerabilityRequest{
		Id:          id,
		Expired:     expired,
		LastUpdated: timestampNowMinus(age),
	}
}
