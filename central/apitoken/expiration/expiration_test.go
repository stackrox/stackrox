//go:build sql_integration

package expiration

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	apiTokenDataStore "github.com/stackrox/rox/central/apitoken/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestAPITokenExpirationNotifier(t *testing.T) {
	suite.Run(t, new(apiTokenExpirationNotifierTestSuite))
}

type apiTokenExpirationNotifierTestSuite struct {
	suite.Suite

	testpostgres *pgtest.TestPostgres

	datastore apiTokenDataStore.DataStore
	notifier  *expirationNotifierImpl

	ctx context.Context
}

func (s *apiTokenExpirationNotifierTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
}

func (s *apiTokenExpirationNotifierTestSuite) SetupTest() {
	s.testpostgres = pgtest.ForT(s.T())
	s.datastore = apiTokenDataStore.NewPostgres(s.testpostgres.DB)
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
	revoked bool) *storage.TokenMetadata {
	var protoNow *types.Timestamp
	var protoExpiration *types.Timestamp
	if now != nil {
		protoNow = protoconv.ConvertTimeToTimestamp(*now)
		protoNow = truncateToMicroSeconds(protoNow)
	}
	if expiration != nil {
		protoExpiration = protoconv.ConvertTimeToTimestamp(*expiration)
		protoExpiration = truncateToMicroSeconds(protoExpiration)
	}
	return &storage.TokenMetadata{
		Id:         uuid.NewV4().String(),
		Name:       "Generated Test Token",
		Roles:      []string{"Admin"},
		IssuedAt:   protoNow,
		Expiration: protoExpiration,
		Revoked:    revoked,
	}
}

func (s *apiTokenExpirationNotifierTestSuite) TestSelectTokenAboutToExpire() {
	now := time.Now()
	expiration := now.Add(2 * time.Hour)
	expiresUntil := now.Add(5 * time.Hour)
	token := generateToken(&now, &expiration, false)
	s.Require().NoError(s.datastore.AddToken(s.ctx, token))

	fetchedTokens, err := s.notifier.listItemsToNotify(now, expiresUntil)
	s.NoError(err)
	expectedResult := token
	expectedResults := []*storage.TokenMetadata{expectedResult}
	s.ElementsMatch(expectedResults, fetchedTokens)
}

func (s *apiTokenExpirationNotifierTestSuite) TestDontSelectTokenNotAboutToExpire() {
	now := time.Now()
	expiration := now.Add(7 * time.Hour)
	expiresUntil := now.Add(5 * time.Hour)
	token := generateToken(&now, &expiration, false)
	s.Require().NoError(s.datastore.AddToken(s.ctx, token))

	fetchedTokens, err := s.notifier.listItemsToNotify(now, expiresUntil)
	s.NoError(err)
	expectedResults := []*storage.TokenMetadata{}
	s.ElementsMatch(expectedResults, fetchedTokens)
}

func (s *apiTokenExpirationNotifierTestSuite) TestDontSelectRevokedToken() {
	now := time.Now()
	expiration := now.Add(2 * time.Hour)
	expiresUntil := now.Add(5 * time.Hour)
	token := generateToken(&now, &expiration, true)
	s.Require().NoError(s.datastore.AddToken(s.ctx, token))

	fetchedTokens, err := s.notifier.listItemsToNotify(now, expiresUntil)
	s.NoError(err)
	expectedResults := []*storage.TokenMetadata{}
	s.ElementsMatch(expectedResults, fetchedTokens)
}

func (s *apiTokenExpirationNotifierTestSuite) TestDontSelectExpiredToken() {
	now := time.Now()
	expiration := now.Add(-2 * time.Hour)
	expiresUntil := now.Add(5 * time.Hour)
	token := generateToken(&now, &expiration, false)
	s.Require().NoError(s.datastore.AddToken(s.ctx, token))

	fetchedTokens, err := s.notifier.listItemsToNotify(now, expiresUntil)
	s.NoError(err)
	expectedResults := []*storage.TokenMetadata{}
	s.ElementsMatch(expectedResults, fetchedTokens)
}

func (s *apiTokenExpirationNotifierTestSuite) TestLogGeneration() {
	now := time.Now()
	sliceDuration := time.Hour
	sliceName := "hour"

	generated1 := now.Add(-(3*time.Hour + 10*time.Minute))
	expiration1 := now.Add(2*time.Hour - 10*time.Minute)
	token1 := generateToken(&generated1, &expiration1, false)
	log1 := generateExpiringTokenLog(token1, now, sliceDuration, sliceName)
	s.Equal("API Token will expire in less than 2 hours", log1)

	generated2 := now.Add(-(4*time.Hour + 10*time.Minute))
	expiration2 := now.Add(1*time.Hour - 10*time.Minute)
	token2 := generateToken(&generated2, &expiration2, false)
	log2 := generateExpiringTokenLog(token2, now, sliceDuration, sliceName)
	s.Equal("API Token will expire in less than 1 hour", log2)

	generated3 := now.Add(-2 * time.Hour)
	expiration3 := now.Add(3 * time.Hour)
	token3 := generateToken(&generated3, &expiration3, false)
	log3 := generateExpiringTokenLog(token3, now, sliceDuration, sliceName)
	s.Equal("API Token will expire in less than 3 hours", log3)
}
