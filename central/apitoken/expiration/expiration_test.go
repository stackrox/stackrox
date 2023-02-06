package expiration

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	apiTokenDataStore "github.com/stackrox/rox/central/apitoken/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestAPITokenEpirationNotifier(t *testing.T) {
	suite.Run(t, new(apiTokenExpirationNotifierTestSuite))
}

type apiTokenExpirationNotifierTestSuite struct {
	suite.Suite

	testpostgres *pgtest.TestPostgres

	datastore apiTokenDataStore.DataStore
	notifier  *expirationNotifierImpl
}

func (s *apiTokenExpirationNotifierTestSuite) SetupSuite() {
	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		s.T().Skip("Notification of expired API tokens is only supported in Postgres mode.")
	}
}

func (s *apiTokenExpirationNotifierTestSuite) SetupTest() {
	s.testpostgres = pgtest.ForT(s.T())
	s.datastore = apiTokenDataStore.NewPostgres(s.testpostgres.Pool)
	s.notifier = newExpirationNotifier(s.datastore)
}

func (s *apiTokenExpirationNotifierTestSuite) TearDownTest() {
	s.testpostgres.Teardown(s.T())
}

func truncateToMicroSeconds(timestamp *types.Timestamp) *types.Timestamp {
	outputTs := timestamp.Clone()
	outputTs.Nanos = 1000 * (outputTs.Nanos / 1000)
	return outputTs
}

func generateToken(now *time.Time,
	expiration *time.Time,
	expirationNotifiedAt *time.Time,
	revoked bool) *storage.TokenMetadata {
	var protoNow *types.Timestamp
	var protoExpiration *types.Timestamp
	var protoExpirationNotifiedAt *types.Timestamp
	if now != nil {
		protoNow = protoconv.ConvertTimeToTimestamp(*now)
		protoNow = truncateToMicroSeconds(protoNow)
	}
	if expiration != nil {
		protoExpiration = protoconv.ConvertTimeToTimestamp(*expiration)
		protoExpiration = truncateToMicroSeconds(protoExpiration)
	}
	if expirationNotifiedAt != nil {
		protoExpirationNotifiedAt = protoconv.ConvertTimeToTimestamp(*expirationNotifiedAt)
		protoExpirationNotifiedAt = truncateToMicroSeconds(protoExpirationNotifiedAt)
	}
	return &storage.TokenMetadata{
		Id:                   uuid.NewV4().String(),
		Name:                 "Generated Test Token",
		Roles:                []string{"Admin"},
		IssuedAt:             protoNow,
		Expiration:           protoExpiration,
		ExpirationNotifiedAt: protoExpirationNotifiedAt,
		Revoked:              revoked,
	}
}

func (s *apiTokenExpirationNotifierTestSuite) TestSelectTokenAboutToExpireNotNotifiedYet() {
	ctx := sac.WithAllAccess(context.Background())
	now := time.Now()
	expiration := now.Add(2 * time.Hour)
	expiresUntil := now.Add(5 * time.Hour)
	notifiedUntil := now.Add(-2 * time.Hour)
	notifiedAt := now.Add(-4 * time.Hour)
	token := generateToken(&now, &expiration, &notifiedAt, false)
	// log.Info("Storing token with ID ", token.GetId(), " notified at ", notifiedAt.String())
	s.Require().NoError(s.datastore.AddToken(ctx, token))

	fetchedTokens, err := s.notifier.listItemsToNotify(now, expiresUntil, notifiedUntil)
	s.NoError(err)
	expectedResult := search.Result{
		ID:      token.GetId(),
		Matches: make(map[string][]string, 0),
		Fields:  nil,
	}
	expectedResults := []search.Result{expectedResult}
	s.ElementsMatch(expectedResults, fetchedTokens)
}

func (s *apiTokenExpirationNotifierTestSuite) TestDontSelectTokenNotAboutToExpire() {
	ctx := sac.WithAllAccess(context.Background())
	now := time.Now()
	expiration := now.Add(7 * time.Hour)
	expiresUntil := now.Add(5 * time.Hour)
	notifiedUntil := now.Add(-2 * time.Hour)
	notifiedAt := now.Add(-4 * time.Hour)
	token := generateToken(&now, &expiration, &notifiedAt, false)
	log.Info("Storing token with ID ", token.GetId(), " notified at ", notifiedAt.String())
	s.Require().NoError(s.datastore.AddToken(ctx, token))

	fetchedTokens, err := s.notifier.listItemsToNotify(now, expiresUntil, notifiedUntil)
	s.NoError(err)
	expectedResults := []search.Result{}
	s.ElementsMatch(expectedResults, fetchedTokens)
}

func (s *apiTokenExpirationNotifierTestSuite) TestDontSelectTokenAboutToExpireNotifiedRecently() {
	ctx := sac.WithAllAccess(context.Background())
	now := time.Now()
	expiration := now.Add(2 * time.Hour)
	expiresUntil := now.Add(5 * time.Hour)
	notifiedUntil := now.Add(-4 * time.Hour)
	notifiedAt := now.Add(-2 * time.Hour)
	token := generateToken(&now, &expiration, &notifiedAt, false)
	log.Info("Storing token with ID ", token.GetId(), " notified at ", notifiedAt.String())
	s.Require().NoError(s.datastore.AddToken(ctx, token))

	fetchedTokens, err := s.notifier.listItemsToNotify(now, expiresUntil, notifiedUntil)
	s.NoError(err)
	expectedResults := []search.Result{}
	s.ElementsMatch(expectedResults, fetchedTokens)
}

func (s *apiTokenExpirationNotifierTestSuite) TestDontSelectRevokedToken() {
	ctx := sac.WithAllAccess(context.Background())
	now := time.Now()
	expiration := now.Add(2 * time.Hour)
	expiresUntil := now.Add(5 * time.Hour)
	notifiedUntil := now.Add(-2 * time.Hour)
	notifiedAt := now.Add(-4 * time.Hour)
	token := generateToken(&now, &expiration, &notifiedAt, true)
	log.Info("Storing token with ID ", token.GetId())
	s.Require().NoError(s.datastore.AddToken(ctx, token))

	fetchedTokens, err := s.notifier.listItemsToNotify(now, expiresUntil, notifiedUntil)
	s.NoError(err)
	expectedResults := []search.Result{}
	s.ElementsMatch(expectedResults, fetchedTokens)
}

func (s *apiTokenExpirationNotifierTestSuite) TestDontSelectExpiredToken() {
	ctx := sac.WithAllAccess(context.Background())
	now := time.Now()
	expiration := now.Add(-2 * time.Hour)
	expiresUntil := now.Add(5 * time.Hour)
	notifiedUntil := now.Add(-2 * time.Hour)
	notifiedAt := now.Add(-4 * time.Hour)
	token := generateToken(&now, &expiration, &notifiedAt, false)
	log.Info("Storing token with ID ", token.GetId())
	s.Require().NoError(s.datastore.AddToken(ctx, token))

	fetchedTokens, err := s.notifier.listItemsToNotify(now, expiresUntil, notifiedUntil)
	s.NoError(err)
	expectedResults := []search.Result{}
	s.ElementsMatch(expectedResults, fetchedTokens)
}
