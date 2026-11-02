package service

import (
	"context"
	"testing"
	"time"

	datastoreMocks "github.com/stackrox/rox/central/pod/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type recordingPodExportServer struct {
	grpc.ServerStream
	ctx  context.Context
	pods []*storage.Pod
}

func (s *recordingPodExportServer) Context() context.Context {
	return s.ctx
}

func (s *recordingPodExportServer) Send(response *v1.ExportPodResponse) error {
	s.pods = append(s.pods, response.GetPod())
	return nil
}

func TestExportPodsMinAgeDays(t *testing.T) {
	now := time.Now().UTC()
	startedDaysAgo := func(days int) *timestamppb.Timestamp {
		return timestamppb.New(now.AddDate(0, 0, -days))
	}
	old := startedDaysAgo(31)
	young := startedDaysAgo(29)
	pods := []*storage.Pod{
		{Id: "mixed", Started: old, LiveInstances: []*storage.ContainerInstance{{Started: old}, {Started: young}}},
		{Id: "restarted", Started: startedDaysAgo(90), LiveInstances: []*storage.ContainerInstance{{Started: young}}},
		{Id: "finished", Started: old, LiveInstances: []*storage.ContainerInstance{{Started: old, Finished: young}}},
		{Id: "young", Started: young, LiveInstances: []*storage.ContainerInstance{{Started: young}}},
		{Id: "no start", LiveInstances: []*storage.ContainerInstance{{Started: old}}},
		{Id: "no instances", Started: old},
	}

	tests := map[string]struct {
		request *v1.ExportPodRequest
		wantIDs []string
		wantErr error
	}{
		"no age filter exports all Pods": {
			request: &v1.ExportPodRequest{},
			wantIDs: []string{"mixed", "restarted", "finished", "young", "no start", "no instances"},
		},
		"only Pods created at least 30 days ago are exported": {
			request: &v1.ExportPodRequest{MinAgeDays: 30},
			wantIDs: []string{"mixed", "restarted", "finished", "no instances"},
		},
		"age filter combines with search query": {
			request: &v1.ExportPodRequest{MinAgeDays: 30, Query: "Namespace:example"},
			wantIDs: []string{"mixed", "restarted", "finished", "no instances"},
		},
		"negative age is invalid": {
			request: &v1.ExportPodRequest{MinAgeDays: -1},
			wantErr: errox.InvalidArgs,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			store := datastoreMocks.NewMockDataStore(gomock.NewController(t))
			if test.wantErr == nil {
				query, err := search.ParseQuery(test.request.GetQuery(), search.MatchAllIfEmpty())
				require.NoError(t, err)
				store.EXPECT().WalkByQuery(gomock.Any(), query, gomock.Any()).DoAndReturn(
					func(_ context.Context, _ *v1.Query, fn func(*storage.Pod) error) error {
						for _, pod := range pods {
							if err := fn(pod); err != nil {
								return err
							}
						}
						return nil
					},
				)
			}

			stream := &recordingPodExportServer{ctx: context.Background()}
			err := (&serviceImpl{datastore: store}).ExportPods(test.request, stream)
			if test.wantErr != nil {
				require.ErrorIs(t, err, test.wantErr)
				return
			}
			require.NoError(t, err)

			var gotIDs []string
			for _, pod := range stream.pods {
				gotIDs = append(gotIDs, pod.GetId())
			}
			require.Equal(t, test.wantIDs, gotIDs)
			if test.request.GetMinAgeDays() > 0 {
				require.Len(t, stream.pods[0].GetLiveInstances(), 2)
			}
		})
	}
}
