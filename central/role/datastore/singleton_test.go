package datastore

import (
	"errors"
	"testing"

	storeMocks "github.com/stackrox/rox/central/role/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSeedDefaultPermissionSets(t *testing.T) {
	for name, tc := range map[string]struct {
		consoleErr error
		otherErr   error
	}{
		"success": {},
		"console name collision": {
			consoleErr: errox.AlreadyExists.New("name collision"),
		},
		"console storage error": {
			consoleErr: errors.New("database unavailable"),
		},
		"other default failure": {
			otherErr: errors.New("database unavailable"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := t.Context()
			store := storeMocks.NewMockPermissionSetStore(gomock.NewController(t))
			console := getDefaultPermissionSet(accesscontrol.ConsoleProxy)
			other := getDefaultPermissionSet(accesscontrol.Admin)
			store.EXPECT().UpsertMany(ctx, []*storage.PermissionSet{other}).Return(tc.otherErr)
			if tc.otherErr == nil {
				store.EXPECT().Upsert(ctx, console).Return(tc.consoleErr)
			}
			err := seedDefaultPermissionSets(ctx, store, []*storage.PermissionSet{console, other})
			if tc.otherErr != nil {
				require.ErrorIs(t, err, tc.otherErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, accesscontrol.DefaultPermissionSetIDs[accesscontrol.ConsoleProxy], console.GetId())
		})
	}
}
