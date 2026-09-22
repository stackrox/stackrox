//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	pgStore "github.com/stackrox/rox/central/namespace/datastore/internal/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
)

func TestGetNamespacesForSACReflectsExternalChanges(t *testing.T) {
	db := pgtest.ForT(t)
	reader := &datastoreImpl{store: pgStore.New(db.DB)}
	writer := pgStore.New(db.DB)
	ctx := sac.WithAllAccess(context.Background())
	namespace := &storage.NamespaceMetadata{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),
		Name:      "namespace",
		Labels:    map[string]string{"team": "old"},
	}

	namespaces, err := reader.GetNamespacesForSAC(context.Background())
	require.NoError(t, err)
	require.Empty(t, namespaces)

	// A second store instance models a write from another Central process.
	require.NoError(t, writer.Upsert(ctx, namespace))
	namespaces, err = reader.GetNamespacesForSAC(context.Background())
	require.NoError(t, err)
	require.Len(t, namespaces, 1)
	require.Equal(t, namespace.GetId(), namespaces[0].GetId())
	require.Equal(t, namespace.GetLabels(), namespaces[0].GetLabels())

	namespace.Labels["team"] = "new"
	require.NoError(t, writer.Upsert(ctx, namespace))
	namespaces, err = reader.GetNamespacesForSAC(context.Background())
	require.NoError(t, err)
	require.Len(t, namespaces, 1)
	require.Equal(t, namespace.GetLabels(), namespaces[0].GetLabels())

	require.NoError(t, writer.Delete(ctx, namespace.GetId()))
	namespaces, err = reader.GetNamespacesForSAC(context.Background())
	require.NoError(t, err)
	require.Empty(t, namespaces)
}

func TestGetNamespacesForSACPropagatesCancellation(t *testing.T) {
	db := pgtest.ForT(t)
	reader := &datastoreImpl{store: pgStore.New(db.DB)}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	namespaces, err := reader.GetNamespacesForSAC(ctx)
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, namespaces)
}
