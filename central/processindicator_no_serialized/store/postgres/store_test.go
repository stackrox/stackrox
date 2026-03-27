//go:build sql_integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ProcessIndicatorNoSerializedsStoreSuite struct {
	suite.Suite
	store  Store
	testDB *pgtest.TestPostgres
}

func TestProcessIndicatorNoSerializedsStore(t *testing.T) {
	suite.Run(t, new(ProcessIndicatorNoSerializedsStoreSuite))
}

func (s *ProcessIndicatorNoSerializedsStoreSuite) SetupSuite() {
	s.testDB = pgtest.ForT(s.T())
	s.store = New(s.testDB.DB)
}

func (s *ProcessIndicatorNoSerializedsStoreSuite) SetupTest() {
	ctx := sac.WithAllAccess(context.Background())
	tag, err := s.testDB.Exec(ctx, "TRUNCATE process_indicator_no_serializeds CASCADE")
	s.T().Log("process_indicator_no_serializeds", tag)
	s.store = New(s.testDB.DB)
	s.NoError(err)
}

func truncateTimestamp(ts *timestamppb.Timestamp) *timestamppb.Timestamp {
	if ts == nil {
		return nil
	}
	t := ts.AsTime().Truncate(time.Microsecond)
	return protocompat.ConvertTimeToTimestampOrNil(&t)
}

func normalizeTimestamps(obj *storage.ProcessIndicatorNoSerialized) {
	obj.ContainerStartTime = truncateTimestamp(obj.ContainerStartTime)
	if obj.Signal != nil {
		obj.Signal.Time = truncateTimestamp(obj.Signal.Time)
	}
}

// withoutChildren returns a clone with child repeated fields cleared,
// for comparing against store reads which don't fetch children by default.
func withoutChildren(obj *storage.ProcessIndicatorNoSerialized) *storage.ProcessIndicatorNoSerialized {
	cloned := proto.Clone(obj).(*storage.ProcessIndicatorNoSerialized)
	if cloned.Signal != nil {
		cloned.Signal.LineageInfo = nil
	}
	return cloned
}

func (s *ProcessIndicatorNoSerializedsStoreSuite) TestStore() {
	ctx := sac.WithAllAccess(context.Background())
	store := s.store

	obj := &storage.ProcessIndicatorNoSerialized{}
	s.NoError(testutils.FullInit(obj, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))
	normalizeTimestamps(obj)

	// Not found before upsert
	found, exists, err := store.Get(ctx, obj.GetId())
	s.NoError(err)
	s.False(exists)
	s.Nil(found)

	withNoAccessCtx := sac.WithNoAccess(ctx)

	// Upsert and read back — children NOT included by default
	s.NoError(store.Upsert(ctx, obj))
	found, exists, err = store.Get(ctx, obj.GetId())
	s.NoError(err)
	s.True(exists)
	protoassert.Equal(s.T(), withoutChildren(obj), found)

	// Count
	count, err := store.Count(ctx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(1, count)
	count, err = store.Count(withNoAccessCtx, search.EmptyQuery())
	s.NoError(err)
	s.Zero(count)

	// Exists
	exists, err = store.Exists(ctx, obj.GetId())
	s.NoError(err)
	s.True(exists)

	// SAC enforcement
	s.NoError(store.Upsert(ctx, obj))
	s.ErrorIs(store.Upsert(withNoAccessCtx, obj), sac.ErrResourceAccessDenied)

	// Delete
	s.NoError(store.Delete(ctx, obj.GetId()))
	found, exists, err = store.Get(ctx, obj.GetId())
	s.NoError(err)
	s.False(exists)
	s.Nil(found)

	// Batch upsert + GetMany
	var objs []*storage.ProcessIndicatorNoSerialized
	var ids []string
	for i := 0; i < 200; i++ {
		o := &storage.ProcessIndicatorNoSerialized{}
		s.NoError(testutils.FullInit(o, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		normalizeTimestamps(o)
		objs = append(objs, o)
		ids = append(ids, o.GetId())
	}

	s.NoError(store.UpsertMany(ctx, objs))

	foundObjs, missing, err := store.GetMany(ctx, ids)
	s.NoError(err)
	s.Empty(missing)
	// Compare without children since they're not fetched by default
	expected := make([]*storage.ProcessIndicatorNoSerialized, len(objs))
	for i, o := range objs {
		expected[i] = withoutChildren(o)
	}
	protoassert.ElementsMatch(s.T(), expected, foundObjs)

	count, err = store.Count(ctx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(200, count)

	s.NoError(store.DeleteMany(ctx, ids))
	count, err = store.Count(ctx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(0, count)
}

func (s *ProcessIndicatorNoSerializedsStoreSuite) TestFetchChildrenOptIn() {
	ctx := sac.WithAllAccess(context.Background())
	store := s.store

	obj := &storage.ProcessIndicatorNoSerialized{}
	s.NoError(testutils.FullInit(obj, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))
	normalizeTimestamps(obj)
	s.NoError(store.Upsert(ctx, obj))

	// Read without children
	found, exists, err := store.Get(ctx, obj.GetId())
	s.NoError(err)
	s.True(exists)
	s.Nil(found.GetSignal().GetLineageInfo(), "children should be nil without explicit fetch")

	// Opt-in: fetch children explicitly
	s.NoError(FetchChildren(ctx, s.testDB.DB, []*storage.ProcessIndicatorNoSerialized{found}))
	protoassert.Equal(s.T(), obj, found)
	s.Len(found.GetSignal().GetLineageInfo(), len(obj.GetSignal().GetLineageInfo()),
		"children should be populated after FetchChildren")

	// Batch: upsert many, GetMany without children, then FetchChildren
	var objs []*storage.ProcessIndicatorNoSerialized
	var ids []string
	for i := 0; i < 10; i++ {
		o := &storage.ProcessIndicatorNoSerialized{}
		s.NoError(testutils.FullInit(o, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		normalizeTimestamps(o)
		objs = append(objs, o)
		ids = append(ids, o.GetId())
	}
	s.NoError(store.UpsertMany(ctx, objs))

	foundObjs, missing, err := store.GetMany(ctx, ids)
	s.NoError(err)
	s.Empty(missing)

	// Before FetchChildren: no child data
	for _, f := range foundObjs {
		s.Nil(f.GetSignal().GetLineageInfo())
	}

	// After FetchChildren: child data populated
	s.NoError(FetchChildren(ctx, s.testDB.DB, foundObjs))
	protoassert.ElementsMatch(s.T(), objs, foundObjs)
}
