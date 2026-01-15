//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	pgStore "github.com/stackrox/rox/central/cve/cluster/datastore/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestClusterCVEDataStoreWithPostgres(t *testing.T) {
	suite.Run(t, new(ClusterCVEPostgresDataStoreTestSuite))
}

type ClusterCVEPostgresDataStoreTestSuite struct {
	suite.Suite

	ctx       context.Context
	db        postgres.DB
	datastore DataStore
}

func (s *ClusterCVEPostgresDataStoreTestSuite) SetupSuite() {
	s.ctx = context.Background()
}

func (s *ClusterCVEPostgresDataStoreTestSuite) SetupTest() {
	s.db = pgtest.ForT(s.T())

	store := pgStore.NewFullStore(s.db)
	ds, err := New(store)
	s.Require().NoError(err)
	s.datastore = ds
}

func (s *ClusterCVEPostgresDataStoreTestSuite) TearDownTest() {
	// Clean up test data
	_, _ = s.db.Exec(context.Background(), "TRUNCATE TABLE cluster_cves CASCADE")
}

func (s *ClusterCVEPostgresDataStoreTestSuite) TearDownSuite() {
	s.db.Close()
}

func (s *ClusterCVEPostgresDataStoreTestSuite) TestSearchClusterCVEs() {
	ctx := sac.WithAllAccess(context.Background())

	cve1 := &storage.ClusterCVE{
		Id:   uuid.NewV4().String(),
		Type: storage.CVE_K8S_CVE,
		CveBaseInfo: &storage.CVEInfo{
			Cve: "CVE-2021-1234",
		},
		Cvss:        8.5,
		Severity:    storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
		ImpactScore: 5.9,
	}

	cve2 := &storage.ClusterCVE{
		Id:   uuid.NewV4().String(),
		Type: storage.CVE_K8S_CVE,
		CveBaseInfo: &storage.CVEInfo{
			Cve: "CVE-2021-5678",
		},
		Cvss:        7.2,
		Severity:    storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
		ImpactScore: 4.5,
	}

	cve3 := &storage.ClusterCVE{
		Id:   uuid.NewV4().String(),
		Type: storage.CVE_OPENSHIFT_CVE,
		CveBaseInfo: &storage.CVEInfo{
			Cve: "CVE-2021-9999",
		},
		Cvss:        5.0,
		Severity:    storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
		ImpactScore: 3.0,
	}

	// Upsert CVEs directly to database
	store := pgStore.NewFullStore(s.db)
	err := store.UpsertMany(ctx, []*storage.ClusterCVE{cve1, cve2, cve3})
	s.NoError(err)

	testCases := []struct {
		name          string
		query         *v1.Query
		expectedCount int
		expectedNames []string
	}{
		{
			name:          "empty query returns all CVEs with names populated",
			query:         pkgSearch.EmptyQuery(),
			expectedCount: 3,
			expectedNames: []string{"CVE-2021-1234", "CVE-2021-5678", "CVE-2021-9999"},
		},
		{
			name:          "nil query defaults to empty query",
			query:         nil,
			expectedCount: 3,
			expectedNames: []string{"CVE-2021-1234", "CVE-2021-5678", "CVE-2021-9999"},
		},
		{
			name:          "query by CVE name",
			query:         pkgSearch.NewQueryBuilder().AddStrings(pkgSearch.CVE, "CVE-2021-1234").ProtoQuery(),
			expectedCount: 1,
			expectedNames: []string{"CVE-2021-1234"},
		},
		{
			name:          "query by severity",
			query:         pkgSearch.NewQueryBuilder().AddStrings(pkgSearch.Severity, storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY.String()).ProtoQuery(),
			expectedCount: 1,
			expectedNames: []string{"CVE-2021-1234"},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			results, err := s.datastore.SearchClusterCVEs(ctx, tc.query)
			s.NoError(err)
			s.Len(results, tc.expectedCount, "Expected %d results, got %d", tc.expectedCount, len(results))

			actualNames := make([]string, 0, len(results))
			for _, result := range results {
				actualNames = append(actualNames, result.GetName())
				s.Equal(v1.SearchCategory_CLUSTER_VULNERABILITIES, result.GetCategory())
				s.NotEmpty(result.GetId())
			}

			if len(tc.expectedNames) > 0 {
				s.ElementsMatch(tc.expectedNames, actualNames)
			}
		})
	}
}
