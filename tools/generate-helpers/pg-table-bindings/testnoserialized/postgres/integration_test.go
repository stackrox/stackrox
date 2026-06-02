//go:build sql_integration

package postgres

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

type IntegrationTestSuite struct {
	suite.Suite
	store  Store
	testDB *pgtest.TestPostgres
	ctx    context.Context
}

func TestIntegration(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.testDB = pgtest.ForT(s.T())
	s.store = CreateTableAndNewStore(s.ctx, s.testDB.DB, s.testDB.GetGormDB(s.T()))
}

func (s *IntegrationTestSuite) SetupTest() {
	tag, err := s.testDB.Exec(s.ctx, "TRUNCATE test_no_serialized_objs CASCADE")
	s.NoError(err)
	s.NotZero(tag)
	s.store = New(s.testDB.DB)
}

func makeTestObj(id string) *storage.TestNoSerializedObj {
	return &storage.TestNoSerializedObj{
		Id:          id,
		Name:        "test-" + id[:8],
		ValueInt32:  42,
		ValueInt64:  123456789,
		ValueUint32: 100,
		ValueUint64: 9999999999,
		ValueBool:   true,
		ValueFloat:  3.14,
		ValueEnum:   storage.TestNoSerializedEnum_TEST_NO_SERIALIZED_ENUM_ACTIVE,
		CreatedAt:   timestamppb.New(time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)),
		Nested: &storage.TestNoSerializedNested{
			Label: "nested-label",
			Score: 999,
		},
		Tags: []string{"tag1", "tag2", "tag3"},
		Metadata: []*storage.TestNoSerializedMetadata{
			{Key: "env", Value: "prod"},
			{Key: "team", Value: "core"},
		},
	}
}

func (s *IntegrationTestSuite) TestRoundTripAllFieldTypes() {
	id := uuid.NewV4().String()
	obj := makeTestObj(id)

	s.NoError(s.store.Upsert(s.ctx, obj))

	got, exists, err := s.store.Get(s.ctx, id)
	s.NoError(err)
	s.True(exists)

	s.Equal(id, got.GetId())
	s.Equal("test-"+id[:8], got.GetName())
	s.Equal(int32(42), got.GetValueInt32())
	s.Equal(int64(123456789), got.GetValueInt64())
	s.Equal(uint32(100), got.GetValueUint32())
	s.Equal(uint64(9999999999), got.GetValueUint64())
	s.True(got.GetValueBool())
	s.InDelta(3.14, float64(got.GetValueFloat()), 0.001)
	s.Equal(storage.TestNoSerializedEnum_TEST_NO_SERIALIZED_ENUM_ACTIVE, got.GetValueEnum())
	s.Equal("nested-label", got.GetNested().GetLabel())
	s.Equal(int64(999), got.GetNested().GetScore())
	s.Equal([]string{"tag1", "tag2", "tag3"}, got.GetTags())
}

func (s *IntegrationTestSuite) TestTimestampPrecision() {
	id := uuid.NewV4().String()
	ts := time.Date(2026, 6, 15, 14, 30, 45, 0, time.UTC)
	obj := makeTestObj(id)
	obj.CreatedAt = timestamppb.New(ts)

	s.NoError(s.store.Upsert(s.ctx, obj))

	got, exists, err := s.store.Get(s.ctx, id)
	s.NoError(err)
	s.True(exists)
	s.True(got.GetCreatedAt().AsTime().Equal(ts))
}

func (s *IntegrationTestSuite) TestMessageBytesRoundTrip() {
	id := uuid.NewV4().String()
	obj := makeTestObj(id)
	obj.Metadata = []*storage.TestNoSerializedMetadata{
		{Key: "k1", Value: "v1"},
		{Key: "k2", Value: "v2"},
		{Key: "k3", Value: "v3"},
	}

	s.NoError(s.store.Upsert(s.ctx, obj))

	got, exists, err := s.store.Get(s.ctx, id)
	s.NoError(err)
	s.True(exists)
	s.Require().Len(got.GetMetadata(), 3)
	s.Equal("k1", got.GetMetadata()[0].GetKey())
	s.Equal("v1", got.GetMetadata()[0].GetValue())
	s.Equal("k3", got.GetMetadata()[2].GetKey())
}

func (s *IntegrationTestSuite) TestEmptyRepeatedFields() {
	id := uuid.NewV4().String()
	obj := makeTestObj(id)
	obj.Tags = nil
	obj.Metadata = nil

	s.NoError(s.store.Upsert(s.ctx, obj))

	got, exists, err := s.store.Get(s.ctx, id)
	s.NoError(err)
	s.True(exists)
	s.Empty(got.GetTags())
	s.Empty(got.GetMetadata())
}

func (s *IntegrationTestSuite) TestEnumValues() {
	for _, enumVal := range []storage.TestNoSerializedEnum{
		storage.TestNoSerializedEnum_TEST_NO_SERIALIZED_ENUM_UNKNOWN,
		storage.TestNoSerializedEnum_TEST_NO_SERIALIZED_ENUM_ACTIVE,
		storage.TestNoSerializedEnum_TEST_NO_SERIALIZED_ENUM_INACTIVE,
	} {
		id := uuid.NewV4().String()
		obj := makeTestObj(id)
		obj.ValueEnum = enumVal

		s.NoError(s.store.Upsert(s.ctx, obj))

		got, _, err := s.store.Get(s.ctx, id)
		s.NoError(err)
		s.Equal(enumVal, got.GetValueEnum())
	}
}

func (s *IntegrationTestSuite) TestSearchByName() {
	for i := 0; i < 5; i++ {
		obj := makeTestObj(uuid.NewV4().String())
		obj.Name = fmt.Sprintf("search-test-%d", i)
		s.NoError(s.store.Upsert(s.ctx, obj))
	}

	q := search.NewQueryBuilder().AddStrings(search.TestNSName, "search-test-2").ProtoQuery()
	results, err := s.store.Search(s.ctx, q)
	s.NoError(err)
	s.Len(results, 1)
}

func (s *IntegrationTestSuite) TestCount() {
	for i := 0; i < 10; i++ {
		s.NoError(s.store.Upsert(s.ctx, makeTestObj(uuid.NewV4().String())))
	}

	count, err := s.store.Count(s.ctx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(10, count)
}

func (s *IntegrationTestSuite) TestWalk() {
	for i := 0; i < 5; i++ {
		s.NoError(s.store.Upsert(s.ctx, makeTestObj(uuid.NewV4().String())))
	}

	var walked int
	err := s.store.Walk(s.ctx, func(obj *storage.TestNoSerializedObj) error {
		walked++
		return nil
	})
	s.NoError(err)
	s.Equal(5, walked)
}

func (s *IntegrationTestSuite) TestGetManyWithMissing() {
	id1 := uuid.NewV4().String()
	id2 := uuid.NewV4().String()
	idMissing := uuid.NewV4().String()

	s.NoError(s.store.Upsert(s.ctx, makeTestObj(id1)))
	s.NoError(s.store.Upsert(s.ctx, makeTestObj(id2)))

	results, missingIndices, err := s.store.GetMany(s.ctx, []string{id1, idMissing, id2})
	s.NoError(err)
	s.Len(results, 2)
	s.Equal([]int{1}, missingIndices)
}

func (s *IntegrationTestSuite) TestUpsertOverwrite() {
	id := uuid.NewV4().String()
	obj := makeTestObj(id)
	obj.Name = "original"
	s.NoError(s.store.Upsert(s.ctx, obj))

	obj.Name = "updated"
	obj.ValueInt32 = 99
	s.NoError(s.store.Upsert(s.ctx, obj))

	got, _, err := s.store.Get(s.ctx, id)
	s.NoError(err)
	s.Equal("updated", got.GetName())
	s.Equal(int32(99), got.GetValueInt32())
}

func (s *IntegrationTestSuite) TestDeleteAndVerify() {
	id := uuid.NewV4().String()
	s.NoError(s.store.Upsert(s.ctx, makeTestObj(id)))

	s.NoError(s.store.Delete(s.ctx, id))

	_, exists, err := s.store.Get(s.ctx, id)
	s.NoError(err)
	s.False(exists)
}

func (s *IntegrationTestSuite) TestUpsertMany() {
	objs := make([]*storage.TestNoSerializedObj, 50)
	for i := range objs {
		objs[i] = makeTestObj(uuid.NewV4().String())
	}

	s.NoError(s.store.UpsertMany(s.ctx, objs))

	count, err := s.store.Count(s.ctx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(50, count)
}

func (s *IntegrationTestSuite) TestConcurrentUpsertAndRead() {
	id := uuid.NewV4().String()
	s.NoError(s.store.Upsert(s.ctx, makeTestObj(id)))

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			obj := makeTestObj(uuid.NewV4().String())
			assert.NoError(s.T(), s.store.Upsert(s.ctx, obj))
		}()
		go func() {
			defer wg.Done()
			_, _, err := s.store.Get(s.ctx, id)
			assert.NoError(s.T(), err)
		}()
	}
	wg.Wait()
}

func (s *IntegrationTestSuite) TestZeroValues() {
	id := uuid.NewV4().String()
	obj := &storage.TestNoSerializedObj{
		Id: id,
	}

	s.NoError(s.store.Upsert(s.ctx, obj))

	got, exists, err := s.store.Get(s.ctx, id)
	s.NoError(err)
	s.True(exists)
	s.Equal("", got.GetName())
	s.Equal(int32(0), got.GetValueInt32())
	s.Equal(false, got.GetValueBool())
	s.Equal(storage.TestNoSerializedEnum_TEST_NO_SERIALIZED_ENUM_UNKNOWN, got.GetValueEnum())
	s.Nil(got.GetCreatedAt())
}

// Suppress unused import warnings — these are used in generated code
var (
	_ = v1.SearchCategory_TEST_NO_SERIALIZED
	_ *gorm.DB
)
