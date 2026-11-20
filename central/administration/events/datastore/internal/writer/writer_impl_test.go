package writer

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubStore struct {
	upserted []*storage.AdministrationEvent
}

func (s *stubStore) Count(context.Context, *v1.Query) (int, error) {
	return 0, nil
}

func (s *stubStore) Get(context.Context, string) (*storage.AdministrationEvent, bool, error) {
	return nil, false, nil
}

func (s *stubStore) GetByQuery(context.Context, *v1.Query) ([]*storage.AdministrationEvent, error) {
	return nil, nil
}

func (s *stubStore) GetByQueryFn(context.Context, *v1.Query, func(*storage.AdministrationEvent) error) error {
	return nil
}

func (s *stubStore) UpsertMany(_ context.Context, objs []*storage.AdministrationEvent) error {
	s.upserted = append(s.upserted, objs...)
	return nil
}

func (s *stubStore) DeleteMany(context.Context, []string) error {
	return nil
}

func (s *stubStore) GetMany(_ context.Context, identifiers []string) ([]*storage.AdministrationEvent, []int, error) {
	missing := make([]int, 0, len(identifiers))
	for i := range identifiers {
		missing = append(missing, i)
	}
	return nil, missing, nil
}

func TestDropForResourceIgnoresLaterUpsert(t *testing.T) {
	ctx := t.Context()
	store := &stubStore{}
	w := New(store)

	dropped := fixtures.GetAdministrationEvent()
	dropped.ResourceID = "remove-id"
	kept := fixtures.GetAdministrationEvent()
	kept.ResourceID = "keep-id"

	w.DropForResource(dropped.GetResourceID())
	require.NoError(t, w.Upsert(ctx, dropped))
	require.NoError(t, w.Upsert(ctx, kept))
	require.NoError(t, w.Flush(ctx))

	require.Len(t, store.upserted, 1)
	assert.Equal(t, kept.GetResourceID(), store.upserted[0].GetResource().GetId())
}
