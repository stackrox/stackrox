package searchbasedpolicies

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/dgraph-io/badger"
	"github.com/stackrox/rox/central/globalindex"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	processIndicatorIndex "github.com/stackrox/rox/central/processindicator/index"
	processIndicatorSearch "github.com/stackrox/rox/central/processindicator/search"
	processIndicatorBadgerStore "github.com/stackrox/rox/central/processindicator/store/badger"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/search/options/deployments"
	"github.com/stackrox/rox/pkg/searchbasedpolicies"
	"github.com/stackrox/rox/pkg/searchbasedpolicies/matcher"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func TestDisallowedMapValueWithRegexKey(t *testing.T) {
	suite.Run(t, new(DisallowedMapValueWithRegexKeyTestSuite))
}

type DisallowedMapValueWithRegexKeyTestSuite struct {
	suite.Suite

	bleveIndex bleve.Index
	db         *badger.DB
	dir        string

	testCtx context.Context

	processDataStore processIndicatorDataStore.DataStore
	matcherBuilder   matcher.Builder
	matcher          searchbasedpolicies.Matcher
}

func (s *DisallowedMapValueWithRegexKeyTestSuite) SetupSuite() {
	envIsolator := testutils.NewEnvIsolator(s.T())
	defer envIsolator.RestoreAll()
	envIsolator.Setenv(features.ImageLabelPolicy.EnvVar(), "true")

	var err error
	s.bleveIndex, err = globalindex.TempInitializeIndices("")
	s.Require().NoError(err)

	s.db, s.dir, err = badgerhelper.NewTemp("default_policies_test.db")
	s.Require().NoError(err)

	processStore := processIndicatorBadgerStore.New(s.db)
	processIndexer := processIndicatorIndex.New(s.bleveIndex)
	processSearcher := processIndicatorSearch.New(processStore, processIndexer)
	s.processDataStore, err = processIndicatorDataStore.New(processStore, nil, processIndexer, processSearcher, nil)
	s.Require().NoError(err)

	policy := &storage.Policy{
		Id:   "No Monkey Business",
		Name: "No Funny Stuff",
		Fields: &storage.PolicyFields{
			DisallowedImageLabel: &storage.KeyValuePolicy{
				Key:   "joseph.*",
				Value: "rules.*",
			},
		},
	}

	s.matcherBuilder = matcher.NewBuilder(
		matcher.NewRegistry(
			s.processDataStore,
		),
		deployments.OptionsMap,
	)

	s.matcher, err = s.matcherBuilder.ForPolicy(policy)
	s.Require().NoError(err)
}

func (s *DisallowedMapValueWithRegexKeyTestSuite) TearDownSuite() {
	s.NoError(s.bleveIndex.Close())
	testutils.TearDownBadger(s.db, s.dir)
}

func (s *DisallowedMapValueWithRegexKeyTestSuite) TestMatches() {
	matchingImage := &storage.Image{
		Id: "MatchingImage",
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Labels: map[string]string{
					"josephyeeee":    "ruleszzzz",
					"not a matching": "string",
				},
			},
		},
	}

	violations, err := s.matcher.MatchOne(s.testCtx, nil, []*storage.Image{matchingImage}, nil)
	s.NoError(err)
	s.NotEmpty(violations.AlertViolations)
}

func (s *DisallowedMapValueWithRegexKeyTestSuite) TestDoesNotMatch() {
	nonmatchingImage := &storage.Image{
		Id: "NonMatchingImage",
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Labels: map[string]string{
					"abc":            "def",
					"not a matching": "string",
				},
			},
		},
	}

	noLabels := &storage.Image{
		Id: "NoLabels",
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Labels: make(map[string]string),
			},
		},
	}

	noMetadata := &storage.Image{
		Id: "NoMetadata",
	}

	// Test in separate calls so we know which one failed.  The violation doesn't say
	violations, err := s.matcher.MatchOne(s.testCtx, nil, []*storage.Image{nonmatchingImage}, nil)
	s.NoError(err)
	s.Empty(violations.AlertViolations)

	violations, err = s.matcher.MatchOne(s.testCtx, nil, []*storage.Image{noLabels}, nil)
	s.NoError(err)
	s.Empty(violations.AlertViolations)

	violations, err = s.matcher.MatchOne(s.testCtx, nil, []*storage.Image{noMetadata}, nil)
	s.NoError(err)
	s.Empty(violations.AlertViolations)
}
