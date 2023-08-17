package nodeinventory

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scancomponent"
)

func getDummyRisk() *storage.Risk {
	return &storage.Risk{
		Score:   1.0,
		Results: make([]*storage.Risk_Result, 0),
		Subject: &storage.RiskSubject{},
	}
}

type mockNodeScorer struct{}

func (m *mockNodeScorer) Score(_ context.Context, _ *storage.Node) *storage.Risk {
	return getDummyRisk()
}

type mockComponentScorer struct{}

func (m *mockComponentScorer) Score(_ context.Context, _ scancomponent.ScanComponent, _ string) *storage.Risk {
	return getDummyRisk()
}

type mockDeploymentScorer struct{}

func (m *mockDeploymentScorer) Score(_ context.Context, _ *storage.Deployment, _ []*storage.Risk) *storage.Risk {
	return getDummyRisk()
}

type mockImageScorer struct{}

func (m *mockImageScorer) Score(_ context.Context, _ *storage.Image) *storage.Risk {
	return getDummyRisk()
}
