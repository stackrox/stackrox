package reprocessor

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	nodeDatastoreMocks "github.com/stackrox/rox/central/node/datastore/mocks"
	riskManagerMocks "github.com/stackrox/rox/central/risk/manager/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	imageEnricher "github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/images/enricher/mocks"
	nodesEnricherMocks "github.com/stackrox/rox/pkg/nodes/enricher/mocks"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/testutils"
	"go.uber.org/mock/gomock"
)

func Test_loopImpl_reprocessNode(t *testing.T) {
	type args struct {
		id string
	}
	type mocks struct {
		nodes        *nodeDatastoreMocks.MockDataStore
		risk         *riskManagerMocks.MockManager
		nodeEnricher *nodesEnricherMocks.MockNodeEnricher
	}
	tests := []struct {
		name       string
		args       args
		node       *storage.Node
		want       bool
		setUpMocks func(t *testing.T, a *args, m *mocks)
	}{
		{
			name: "when node is RHCOS then nothing is done",
			setUpMocks: func(t *testing.T, a *args, m *mocks) {
				node := &storage.Node{
					OsImage: "Red Hat Enterprise Linux CoreOS 412.86.202302091419-0 (Ootpa)",
				}
				m.nodes.EXPECT().GetNode(gomock.Any(), a.id).Return(node, true, nil)
			},
		},
		{
			name: "when node is not RHCOS then scanner is called and node is upserted",
			setUpMocks: func(t *testing.T, a *args, m *mocks) {
				node := &storage.Node{
					OsImage:     "Something that is not RHCOS",
					LastUpdated: protocompat.TimestampNow(),
				}
				gomock.InOrder(
					m.nodes.EXPECT().GetNode(gomock.Any(), gomock.Eq(a.id)).Times(1).Return(node, true, nil),
					m.nodeEnricher.EXPECT().EnrichNode(node).Times(1).Return(nil),
					m.risk.EXPECT().CalculateRiskAndUpsertNode(gomock.Any()).Return(nil).Times(1),
				)
			},
			want: true,
		},
		{
			name: "when node storage returns err then returns false",
			setUpMocks: func(t *testing.T, a *args, m *mocks) {
				m.nodes.EXPECT().GetNode(gomock.Any(), a.id).Times(1).Return(nil, false, errors.New("foobar"))
			},
			want: false,
		},
		{
			name: "when node storage is successful but node is not found then returns false",
			setUpMocks: func(t *testing.T, a *args, m *mocks) {
				m.nodes.EXPECT().GetNode(gomock.Any(), a.id).Times(1).Return(nil, false, nil)
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := mocks{
				nodes:        nodeDatastoreMocks.NewMockDataStore(ctrl),
				risk:         riskManagerMocks.NewMockManager(ctrl),
				nodeEnricher: nodesEnricherMocks.NewMockNodeEnricher(ctrl),
			}
			tt.setUpMocks(t, &tt.args, &m)
			l := &loopImpl{
				nodes:        m.nodes,
				risk:         m.risk,
				nodeEnricher: m.nodeEnricher,
			}
			if got := l.reprocessNode(tt.args.id); got != tt.want {
				t.Errorf("reprocessNode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReprocessWatchedImageDelegation(t *testing.T) {
	t.Run("delegation disabled", func(t *testing.T) {
		testutils.MustUpdateFeature(t, features.DelegateWatchedImageReprocessing, false)

		enrichmentCtx := gomock.Cond(func(ctxRaw any) bool {
			// Ensure that the enrichment isn't delegable.
			ectx := ctxRaw.(imageEnricher.EnrichmentContext)
			return !ectx.Delegable
		})

		ctrl := gomock.NewController(t)
		enricher := mocks.NewMockImageEnricher(ctrl)
		enricher.EXPECT().EnrichImage(emptyCtx, enrichmentCtx, gomock.Any())

		loop := &loopImpl{imageEnricher: enricher}
		loop.reprocessWatchedImage("example.com/repo/path:tag")
	})

	t.Run("delegation enabled", func(t *testing.T) {
		testutils.MustUpdateFeature(t, features.DelegateWatchedImageReprocessing, true)

		ctx := gomock.Cond(func(ctxRaw any) bool {
			// Delegation will fail if context does not have image read access.
			ctx := ctxRaw.(context.Context)
			scopeChecker := sac.GlobalAccessScopeChecker(ctx).
				AccessMode(storage.Access_READ_ACCESS).
				Resource(resources.Image)

			return scopeChecker.IsAllowed()
		})
		enrichmentCtx := gomock.Cond(func(ctxRaw any) bool {
			// The enrichment must be delegable.
			ectx := ctxRaw.(imageEnricher.EnrichmentContext)
			return ectx.Delegable
		})

		ctrl := gomock.NewController(t)
		enricher := mocks.NewMockImageEnricher(ctrl)
		enricher.EXPECT().EnrichImage(ctx, enrichmentCtx, gomock.Any())

		loop := &loopImpl{imageEnricher: enricher}
		loop.reprocessWatchedImage("example.com/repo/path:tag")
	})
}
