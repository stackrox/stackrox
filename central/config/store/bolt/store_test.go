package bolt

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore(t *testing.T) {
	db, err := bolthelper.NewTemp("config_test.db")
	require.NoError(t, err)

	store := New(db)

	ctx := context.Background()

	config, exists, err := store.Get(ctx)
	require.NoError(t, err)
	assert.False(t, exists)
	assert.Nil(t, config)

	newConfig := &storage.Config{
		PublicConfig: &storage.PublicConfig{
			LoginNotice: &storage.LoginNotice{
				Text: "text",
			},
		},
	}
	assert.NoError(t, store.Upsert(ctx, newConfig))

	config, exists, err = store.Get(ctx)
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, newConfig, config)
}
