package service

import (
	"context"
	"testing"

	aiWorkloadDSMocks "github.com/stackrox/rox/central/aiworkload/datastore/mocks"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetAIWorkload(t *testing.T) {
	tests := map[string]struct {
		setupMocks   func(*aiWorkloadDSMocks.MockDataStore)
		request      *v2.GetAIWorkloadRequest
		expectsError bool
		validate     func(t *testing.T, result *v2.AIWorkload)
	}{
		"returns workload when found": {
			setupMocks: func(ds *aiWorkloadDSMocks.MockDataStore) {
				ds.EXPECT().GetAIWorkload(gomock.Any(), "test-id").Return(&storage.AIWorkload{
					Id:          "test-id",
					Name:        "test-model",
					Namespace:   "ai-project",
					ModelFormat: "vLLM",
				}, true, nil)
			},
			request: &v2.GetAIWorkloadRequest{Id: "test-id"},
			validate: func(t *testing.T, result *v2.AIWorkload) {
				assert.Equal(t, "test-id", result.GetId())
				assert.Equal(t, "test-model", result.GetName())
				assert.Equal(t, "vLLM", result.GetModelFormat())
			},
		},
		"returns not found error when missing": {
			setupMocks: func(ds *aiWorkloadDSMocks.MockDataStore) {
				ds.EXPECT().GetAIWorkload(gomock.Any(), "missing-id").Return(nil, false, nil)
			},
			request:      &v2.GetAIWorkloadRequest{Id: "missing-id"},
			expectsError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			ds := aiWorkloadDSMocks.NewMockDataStore(ctrl)
			tc.setupMocks(ds)

			svc := &serviceImpl{datastore: ds}
			result, err := svc.GetAIWorkload(context.Background(), tc.request)
			if tc.expectsError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				tc.validate(t, result)
			}
		})
	}
}

func TestListAIWorkloads(t *testing.T) {
	tests := map[string]struct {
		setupMocks func(*aiWorkloadDSMocks.MockDataStore)
		request    *v2.ListAIWorkloadsRequest
		validate   func(t *testing.T, result *v2.ListAIWorkloadsResponse)
	}{
		"returns workloads": {
			setupMocks: func(ds *aiWorkloadDSMocks.MockDataStore) {
				ds.EXPECT().CountAIWorkloads(gomock.Any(), gomock.Any()).Return(2, nil)
				ds.EXPECT().SearchRawAIWorkloads(gomock.Any(), gomock.Any()).Return([]*storage.AIWorkload{
					{Id: "1", Name: "model-a", ModelFormat: "vLLM"},
					{Id: "2", Name: "model-b", ModelFormat: "pytorch"},
				}, nil)
			},
			request: &v2.ListAIWorkloadsRequest{},
			validate: func(t *testing.T, result *v2.ListAIWorkloadsResponse) {
				assert.Equal(t, int32(2), result.GetTotalCount())
				require.Len(t, result.GetAiWorkloads(), 2)
				assert.Equal(t, "model-a", result.GetAiWorkloads()[0].GetName())
				assert.Equal(t, "model-b", result.GetAiWorkloads()[1].GetName())
			},
		},
		"returns empty list": {
			setupMocks: func(ds *aiWorkloadDSMocks.MockDataStore) {
				ds.EXPECT().CountAIWorkloads(gomock.Any(), gomock.Any()).Return(0, nil)
				ds.EXPECT().SearchRawAIWorkloads(gomock.Any(), gomock.Any()).Return(nil, nil)
			},
			request: &v2.ListAIWorkloadsRequest{},
			validate: func(t *testing.T, result *v2.ListAIWorkloadsResponse) {
				assert.Equal(t, int32(0), result.GetTotalCount())
				assert.Empty(t, result.GetAiWorkloads())
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			ds := aiWorkloadDSMocks.NewMockDataStore(ctrl)
			tc.setupMocks(ds)

			svc := &serviceImpl{datastore: ds}
			result, err := svc.ListAIWorkloads(context.Background(), tc.request)
			require.NoError(t, err)
			tc.validate(t, result)
		})
	}
}
