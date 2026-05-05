package datastore

import (
	"context"
	"errors"
	"testing"

	"github.com/stackrox/rox/central/auth/m2m"
	"github.com/stackrox/rox/central/auth/m2m/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func Test_wrapRollBackSet(t *testing.T) {
	ctx := context.Background()
	testIssuer := "https://test.example.com"
	originalErr := errors.New("original error")
	removeErr := errors.New("remove error")
	rollbackErr := errors.New("rollback error")

	testConfig := &storage.AuthMachineToMachineConfig{
		Id:     "test-id",
		Issuer: testIssuer,
	}

	storedConfig1 := &storage.AuthMachineToMachineConfig{
		Id:     "stored-1",
		Issuer: testIssuer,
	}

	storedConfig2 := &storage.AuthMachineToMachineConfig{
		Id:     "stored-2",
		Issuer: testIssuer,
	}

	tests := map[string]struct {
		storedConfigs          []*storage.AuthMachineToMachineConfig
		setupExistingExchanger func(*gomock.Controller) m2m.TokenExchanger
		setupMocks             func(*mocks.MockTokenExchangerSet)
		expectedErrorContains  []string
	}{
		"success with stored configs - removes new config and rolls back to stored configs": {
			storedConfigs: []*storage.AuthMachineToMachineConfig{storedConfig1, storedConfig2},
			setupMocks: func(mockSet *mocks.MockTokenExchangerSet) {
				mockSet.EXPECT().RemoveTokenExchanger(ctx, testConfig).Return(nil)
				mockSet.EXPECT().RollbackExchanger(ctx, testIssuer, []*storage.AuthMachineToMachineConfig{storedConfig1, storedConfig2}).Return(nil)
			},
			expectedErrorContains: []string{"original error"},
		},
		"success with existing exchanger configs - removes new config and rolls back to exchanger configs": {
			storedConfigs: nil,
			setupExistingExchanger: func(ctrl *gomock.Controller) m2m.TokenExchanger {
				exchanger := mocks.NewMockTokenExchanger(ctrl)
				exchanger.EXPECT().Configs().Return([]*storage.AuthMachineToMachineConfig{storedConfig1}).AnyTimes()
				return exchanger
			},
			setupMocks: func(mockSet *mocks.MockTokenExchangerSet) {
				mockSet.EXPECT().RemoveTokenExchanger(ctx, testConfig).Return(nil)
				mockSet.EXPECT().RollbackExchanger(ctx, testIssuer, []*storage.AuthMachineToMachineConfig{storedConfig1}).Return(nil)
			},
			expectedErrorContains: []string{"original error"},
		},
		"no stored configs and no existing exchanger - only removes new config": {
			storedConfigs:          nil,
			setupExistingExchanger: nil,
			setupMocks: func(mockSet *mocks.MockTokenExchangerSet) {
				mockSet.EXPECT().RemoveTokenExchanger(ctx, testConfig).Return(nil)
			},
			expectedErrorContains: []string{"original error"},
		},
		"existing exchanger with empty configs - only removes new config": {
			storedConfigs: nil,
			setupExistingExchanger: func(ctrl *gomock.Controller) m2m.TokenExchanger {
				exchanger := mocks.NewMockTokenExchanger(ctrl)
				exchanger.EXPECT().Configs().Return([]*storage.AuthMachineToMachineConfig{}).AnyTimes()
				return exchanger
			},
			setupMocks: func(mockSet *mocks.MockTokenExchangerSet) {
				mockSet.EXPECT().RemoveTokenExchanger(ctx, testConfig).Return(nil)
			},
			expectedErrorContains: []string{"original error"},
		},
		"remove fails with stored configs - joins remove error with rollback result": {
			storedConfigs: []*storage.AuthMachineToMachineConfig{storedConfig1},
			setupMocks: func(mockSet *mocks.MockTokenExchangerSet) {
				mockSet.EXPECT().RemoveTokenExchanger(ctx, testConfig).Return(removeErr)
				mockSet.EXPECT().RollbackExchanger(ctx, testIssuer, []*storage.AuthMachineToMachineConfig{storedConfig1}).Return(nil)
			},
			expectedErrorContains: []string{"rolling back due to", "original error", "remove error"},
		},
		"rollback fails with stored configs - joins rollback error": {
			storedConfigs: []*storage.AuthMachineToMachineConfig{storedConfig1},
			setupMocks: func(mockSet *mocks.MockTokenExchangerSet) {
				mockSet.EXPECT().RemoveTokenExchanger(ctx, testConfig).Return(nil)
				mockSet.EXPECT().RollbackExchanger(ctx, testIssuer, []*storage.AuthMachineToMachineConfig{storedConfig1}).Return(rollbackErr)
			},
			expectedErrorContains: []string{"rolling back due to", "original error", "rollback error"},
		},
		"both remove and rollback fail - joins all errors": {
			storedConfigs: []*storage.AuthMachineToMachineConfig{storedConfig1},
			setupMocks: func(mockSet *mocks.MockTokenExchangerSet) {
				mockSet.EXPECT().RemoveTokenExchanger(ctx, testConfig).Return(removeErr)
				mockSet.EXPECT().RollbackExchanger(ctx, testIssuer, []*storage.AuthMachineToMachineConfig{storedConfig1}).Return(rollbackErr)
			},
			expectedErrorContains: []string{"rolling back due to", "original error", "remove error", "rollback error"},
		},
		"prioritizes stored configs over existing exchanger configs": {
			storedConfigs: []*storage.AuthMachineToMachineConfig{storedConfig1},
			setupExistingExchanger: func(ctrl *gomock.Controller) m2m.TokenExchanger {
				exchanger := mocks.NewMockTokenExchanger(ctrl)
				// Configs() should not be called since we have storedConfigs
				return exchanger
			},
			setupMocks: func(mockSet *mocks.MockTokenExchangerSet) {
				mockSet.EXPECT().RemoveTokenExchanger(ctx, testConfig).Return(nil)
				mockSet.EXPECT().RollbackExchanger(ctx, testIssuer, []*storage.AuthMachineToMachineConfig{storedConfig1}).Return(nil)
			},
			expectedErrorContains: []string{"original error"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			var existingExchanger m2m.TokenExchanger
			if tc.setupExistingExchanger != nil {
				existingExchanger = tc.setupExistingExchanger(ctrl)
			}

			mockSet := mocks.NewMockTokenExchangerSet(ctrl)
			tc.setupMocks(mockSet)

			ds := &datastoreImpl{
				set: mockSet,
			}

			err := ds.wrapRollBackSet(ctx, originalErr, testIssuer, tc.storedConfigs, testConfig, existingExchanger)

			assert.Error(t, err)
			for _, expectedStr := range tc.expectedErrorContains {
				assert.Contains(t, err.Error(), expectedStr)
			}
		})
	}
}
