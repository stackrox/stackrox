//go:build sql_integration

package tests

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stackrox/rox/central/clusterinit/store"
	pgStore "github.com/stackrox/rox/central/clusterinit/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestClusterInitStore(t *testing.T) {
	suite.Run(t, new(clusterInitStoreTestSuite))
}

type clusterInitStoreTestSuite struct {
	suite.Suite
	store store.Store
	ctx   context.Context
	db    postgres.DB
}

func (s *clusterInitStoreTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
}

func (s *clusterInitStoreTestSuite) SetupTest() {
	s.db = pgtest.ForT(s.T())
	s.store = store.NewStore(pgStore.New(s.db))
}

func (s *clusterInitStoreTestSuite) TearDownTest() {
	s.db.Close()
}

func (s *clusterInitStoreTestSuite) TestIDCollisionOnAdd() {
	meta := &storage.InitBundleMeta{
		Id:   "0123456789",
		Name: "test name",
	}
	idCollision := &storage.InitBundleMeta{
		Id:   "0123456789",
		Name: "id collision",
	}

	err := s.store.Add(s.ctx, meta)
	s.NoError(err)

	err = s.store.Add(s.ctx, idCollision)
	s.Error(err)
	s.True(errors.Is(err, store.ErrInitBundleIDCollision))
}

func (s *clusterInitStoreTestSuite) TestNameCollisionOnAdd() {
	meta := &storage.InitBundleMeta{
		Id:   "0123456789",
		Name: "test_name",
	}
	err := s.store.Add(s.ctx, meta)
	s.NoError(err)

	meta2 := &storage.InitBundleMeta{
		Id:   "9876543210",
		Name: "test_name",
	}

	err = s.store.Add(s.ctx, meta2)
	s.Error(err)
}

func (s *clusterInitStoreTestSuite) TestRevokeToken() {
	meta := &storage.InitBundleMeta{
		Id:        "012345",
		Name:      "available",
		IsRevoked: false,
	}
	toRevokeMeta := &storage.InitBundleMeta{
		Id:        "0123456789",
		Name:      "revoked",
		IsRevoked: false,
	}
	toReuseMetaName := &storage.InitBundleMeta{
		Id:        "0123456",
		Name:      "revoked",
		IsRevoked: false,
	}

	for _, m := range []*storage.InitBundleMeta{toRevokeMeta, meta} {
		err := s.store.Add(s.ctx, m)
		s.Require().NoError(err)
	}

	storedMeta, err := s.store.Get(s.ctx, toRevokeMeta.GetId())
	s.Require().NoError(err)
	s.False(storedMeta.GetIsRevoked())

	err = s.store.Revoke(s.ctx, toRevokeMeta.GetId())
	s.Require().NoError(err)

	// test GetAll ignores revoked bundles
	all, err := s.store.GetAll(s.ctx)
	s.Require().NoError(err)
	s.Len(all, 1)
	s.Equal("available", all[0].GetName())

	// test name can be reused after revoking an init-bundle
	err = s.store.Add(s.ctx, toReuseMetaName)
	s.Require().NoError(err)
	reused, err := s.store.Get(s.ctx, toReuseMetaName.GetId())
	s.Require().NoError(err)
	s.Equal(toReuseMetaName.GetName(), reused.GetName())
	s.Equal(toRevokeMeta.GetName(), reused.GetName())
}

// Tests auto revocation for a CRS with maxRegistrations == 0 (unlimited registrations).

func (s *clusterInitStoreTestSuite) TestCrsWithoutMaxRegistrations() {
	clusterName := fmt.Sprintf("some-cluster-%s", uuid.NewV4().String())
	crsId := uuid.NewV4().String()
	crsMeta := &storage.InitBundleMeta{
		Id:        crsId,
		Name:      fmt.Sprintf("test-crs-unlimited-1-%d", rand.Intn(10000)),
		CreatedAt: timestamppb.New(time.Now()),
		Version:   storage.InitBundleMeta_CRS,
	}
	err := s.store.Add(s.ctx, crsMeta)
	s.Require().NoError(err, "adding CRS %s failed", crsId)

	err = s.store.InitiateClusterRegistration(s.ctx, crsId, clusterName)
	s.NoErrorf(err, "recording initiated registration for CRS %s failed", crsId)

	crsMeta, err = s.store.Get(s.ctx, crsMeta.GetId())
	s.NoErrorf(err, "retrieving CRS %s failed", crsId)
	s.Empty(crsMeta.GetRegistrationsInitiated(), "CRS %s has registrationsInitiated non-empty, even though registrations are unlimited", crsId)

	err = s.store.MarkClusterRegistrationComplete(s.ctx, crsId, clusterName)
	s.NoErrorf(err, "recording completed registration for CRS %s failed", crsId)
	s.Empty(crsMeta.GetRegistrationsInitiated(), "CRS %s has registrationsInitiated non-empty, even though registrations are unlimited", crsId)
	s.Empty(crsMeta.GetRegistrationsCompleted(), "CRS %s has registrationsCompleted non-empty, even though registrations are unlimited", crsId)
	s.Falsef(crsMeta.GetIsRevoked(), "CRS %s is revoked", crsId)
}

// Tests auto revocation for a CRS with maxRegistrations == 1.

func (s *clusterInitStoreTestSuite) TestCrsAutoRevocationOneShot() {
	clusterName := fmt.Sprintf("some-cluster-%s", uuid.NewV4().String())
	crsId := uuid.NewV4().String()
	crsMeta := &storage.InitBundleMeta{
		Id:               crsId,
		Name:             fmt.Sprintf("test-crs-auto-revocation-1-%d", rand.Intn(10000)),
		CreatedAt:        timestamppb.New(time.Now()),
		Version:          storage.InitBundleMeta_CRS,
		MaxRegistrations: 1,
	}
	err := s.store.Add(s.ctx, crsMeta)
	s.Require().NoError(err, "adding CRS %s failed", crsId)

	err = s.store.InitiateClusterRegistration(s.ctx, crsId, clusterName)
	s.NoErrorf(err, "recording initiated registration for CRS %s failed", crsId)

	err = s.store.InitiateClusterRegistration(s.ctx, crsId, clusterName)
	s.Error(err, "cluster registration still possible")

	err = s.store.MarkClusterRegistrationComplete(s.ctx, crsId, clusterName)
	s.NoErrorf(err, "recording completed registration for CRS %s failed", crsId)

	crsMeta, err = s.store.Get(s.ctx, crsId)
	s.NoErrorf(err, "receiving CRS %s", crsId)
	s.Truef(crsMeta.GetIsRevoked(), "CRS %s is not revoked", crsId)

}

// Tests auto revocation for a CRS with maxRegistrations > 1.
func (s *clusterInitStoreTestSuite) TestCrsAutoRevocationAfterTwoRegistrations() {
	clusterName := fmt.Sprintf("some-cluster-%s", uuid.NewV4().String())
	crsId := uuid.NewV4().String()
	crsMeta := &storage.InitBundleMeta{
		Id:               crsId,
		Name:             fmt.Sprintf("test-crs-auto-revocation-2-%d", rand.Intn(10000)),
		CreatedAt:        timestamppb.New(time.Now()),
		Version:          storage.InitBundleMeta_CRS,
		MaxRegistrations: 2,
	}
	err := s.store.Add(s.ctx, crsMeta)
	s.Require().NoError(err, "adding CRS %s failed", crsId)

	err = s.store.InitiateClusterRegistration(s.ctx, crsId, clusterName)
	s.NoErrorf(err, "recording initiated registration for CRS %s failed", crsId)

	err = s.store.InitiateClusterRegistration(s.ctx, crsId, clusterName)
	s.NoError(err, "cluster registration not possible")

	err = s.store.MarkClusterRegistrationComplete(s.ctx, crsId, clusterName)
	s.NoErrorf(err, "recording completed registration for CRS %s failed", crsId)

	crsMeta, err = s.store.Get(s.ctx, crsId)
	s.NoErrorf(err, "receiving CRS %s", crsId)
	s.Falsef(crsMeta.GetIsRevoked(), "CRS %s is not revoked", crsId)

	clusterName = fmt.Sprintf("some-cluster-%s", uuid.NewV4().String())

	err = s.store.InitiateClusterRegistration(s.ctx, crsId, clusterName)
	s.NoErrorf(err, "recording initiated registration for CRS %s failed", crsId)

	err = s.store.InitiateClusterRegistration(s.ctx, crsId, clusterName)
	s.Error(err, "cluster registration still possible")

	err = s.store.MarkClusterRegistrationComplete(s.ctx, crsId, clusterName)
	s.NoErrorf(err, "recording completed registration for CRS %s failed", crsId)

	crsMeta, err = s.store.Get(s.ctx, crsId)
	s.NoErrorf(err, "receiving CRS %s", crsId)
	s.Truef(crsMeta.GetIsRevoked(), "CRS %s is not revoked", crsId)
}
