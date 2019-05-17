package store

import (
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

	config, err := store.GetConfig()
	require.NoError(t, err)

	assert.NotNil(t, config)

	newConfig := &storage.Config{
		PublicConfig: &storage.PublicConfig{
			LoginNotice: &storage.LoginNotice{
				Text: "text",
			},
		},
	}
	assert.NoError(t, store.UpdateConfig(newConfig))

	config, err = store.GetConfig()
	assert.NoError(t, err)
	assert.Equal(t, newConfig, config)
}
