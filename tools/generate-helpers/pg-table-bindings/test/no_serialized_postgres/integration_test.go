//go:build sql_integration

package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type NoSerializedIntegrationSuite struct {
	suite.Suite
	store  Store
	testDB *pgtest.TestPostgres
	ctx    context.Context
}

func TestNoSerializedIntegration(t *testing.T) {
	suite.Run(t, new(NoSerializedIntegrationSuite))
}

func (s *NoSerializedIntegrationSuite) SetupSuite() {
	s.testDB = pgtest.ForT(s.T())
	s.store = New(s.testDB.DB)
	s.ctx = sac.WithAllAccess(context.Background())
}

func (s *NoSerializedIntegrationSuite) SetupTest() {
	_, err := s.testDB.Exec(s.ctx, "TRUNCATE test_no_serializeds CASCADE")
	s.Require().NoError(err)
	s.store = New(s.testDB.DB)
}

func (s *NoSerializedIntegrationSuite) TestRoundTrip() {
	obj := s.makeTestObject("round-trip")

	s.Require().NoError(s.store.Upsert(s.ctx, obj))

	got, exists, err := s.store.Get(s.ctx, obj.GetId())
	s.Require().NoError(err)
	s.True(exists)

	protoassert.Equal(s.T(), obj, got)
}

func (s *NoSerializedIntegrationSuite) TestRoundTripAllFieldTypes() {
	obj := s.makeTestObject("field-types")
	obj.Int32Val = -42
	obj.Int64Val = -9999999999
	obj.Uint64Val = 18446744073709551000
	obj.BoolVal = false
	obj.FloatVal = -3.14
	obj.DoubleVal = 2.718281828459045
	obj.Priority = storage.TestNoSerialized_CRITICAL_PRIORITY
	obj.Tags = []string{"alpha", "beta", "gamma"}

	s.Require().NoError(s.store.Upsert(s.ctx, obj))

	got, exists, err := s.store.Get(s.ctx, obj.GetId())
	s.Require().NoError(err)
	s.True(exists)

	s.Equal(obj.GetInt32Val(), got.GetInt32Val())
	s.Equal(obj.GetInt64Val(), got.GetInt64Val())
	s.Equal(obj.GetUint64Val(), got.GetUint64Val())
	s.Equal(obj.GetBoolVal(), got.GetBoolVal())
	s.InDelta(float64(obj.GetFloatVal()), float64(got.GetFloatVal()), 0.001)
	s.InDelta(obj.GetDoubleVal(), got.GetDoubleVal(), 1e-12)
	s.Equal(obj.GetPriority(), got.GetPriority())
	s.Equal(obj.GetTags(), got.GetTags())
	s.Equal(obj.GetMetadata().GetAuthor(), got.GetMetadata().GetAuthor())
	s.Equal(obj.GetMetadata().GetVersion(), got.GetMetadata().GetVersion())
	s.Equal(obj.GetMetadata().GetRevision(), got.GetMetadata().GetRevision())
}

func (s *NoSerializedIntegrationSuite) TestUpsertMany() {
	objs := make([]*storage.TestNoSerialized, 200)
	for i := range objs {
		objs[i] = s.makeTestObject(fmt.Sprintf("batch-%d", i))
	}

	s.Require().NoError(s.store.UpsertMany(s.ctx, objs))

	count, err := s.store.Count(s.ctx, search.EmptyQuery())
	s.Require().NoError(err)
	s.Equal(200, count)
}

func (s *NoSerializedIntegrationSuite) TestGetMany() {
	objs := make([]*storage.TestNoSerialized, 10)
	ids := make([]string, 10)
	for i := range objs {
		objs[i] = s.makeTestObject(fmt.Sprintf("getmany-%d", i))
		ids[i] = objs[i].GetId()
	}
	s.Require().NoError(s.store.UpsertMany(s.ctx, objs))

	got, missing, err := s.store.GetMany(s.ctx, ids)
	s.Require().NoError(err)
	s.Empty(missing)
	s.Len(got, 10)
}

func (s *NoSerializedIntegrationSuite) TestWalk() {
	objs := make([]*storage.TestNoSerialized, 5)
	for i := range objs {
		objs[i] = s.makeTestObject(fmt.Sprintf("walk-%d", i))
	}
	s.Require().NoError(s.store.UpsertMany(s.ctx, objs))

	var walked int
	err := s.store.Walk(s.ctx, func(_ *storage.TestNoSerialized) error {
		walked++
		return nil
	})
	s.Require().NoError(err)
	s.Equal(5, walked)
}

func (s *NoSerializedIntegrationSuite) TestDelete() {
	obj := s.makeTestObject("delete-me")
	s.Require().NoError(s.store.Upsert(s.ctx, obj))

	s.Require().NoError(s.store.Delete(s.ctx, obj.GetId()))

	_, exists, err := s.store.Get(s.ctx, obj.GetId())
	s.Require().NoError(err)
	s.False(exists)
}

func (s *NoSerializedIntegrationSuite) TestDeleteMany() {
	objs := make([]*storage.TestNoSerialized, 5)
	ids := make([]string, 5)
	for i := range objs {
		objs[i] = s.makeTestObject(fmt.Sprintf("delmany-%d", i))
		ids[i] = objs[i].GetId()
	}
	s.Require().NoError(s.store.UpsertMany(s.ctx, objs))

	s.Require().NoError(s.store.DeleteMany(s.ctx, ids))

	count, err := s.store.Count(s.ctx, search.EmptyQuery())
	s.Require().NoError(err)
	s.Equal(0, count)
}

func (s *NoSerializedIntegrationSuite) TestUpsertOverwrite() {
	obj := s.makeTestObject("overwrite")
	s.Require().NoError(s.store.Upsert(s.ctx, obj))

	obj.Name = "updated-name"
	obj.Priority = storage.TestNoSerialized_LOW_PRIORITY
	s.Require().NoError(s.store.Upsert(s.ctx, obj))

	got, _, err := s.store.Get(s.ctx, obj.GetId())
	s.Require().NoError(err)
	s.Equal("updated-name", got.GetName())
	s.Equal(storage.TestNoSerialized_LOW_PRIORITY, got.GetPriority())
}

func (s *NoSerializedIntegrationSuite) makeTestObject(name string) *storage.TestNoSerialized {
	// Use microsecond-precision timestamp (Postgres truncates sub-microsecond)
	now := time.Now().Truncate(time.Microsecond)
	return &storage.TestNoSerialized{
		Id:          uuid.NewV4().String(),
		Name:        name,
		Description: "test description for " + name,
		Int32Val:    42,
		Int64Val:    9999999999,
		Uint64Val:   200,
		BoolVal:     true,
		FloatVal:    3.14,
		DoubleVal:   2.71828,
		Priority:    storage.TestNoSerialized_HIGH_PRIORITY,
		CreatedAt:   timestamppb.New(now),
		ClusterId:   uuid.NewV4().String(),
		Tags:        []string{"tag1", "tag2", "tag3"},
		Metadata: &storage.TestNoSerialized_Metadata{
			Author:   "test-author",
			Version:  "1.0.0",
			Revision: 7,
		},
	}
}
