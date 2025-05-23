package common

import (
	"context"
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/testutils/roletest"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type testGatherer []*dto.MetricFamily

func (g testGatherer) Gather() ([]*dto.MetricFamily, error) {
	return g, nil
}

func makePair(name, value string) *dto.LabelPair {
	return &dto.LabelPair{Name: pointers.String(name), Value: pointers.String(value)}
}

func TestSacGatherer(t *testing.T) {

	clusterOk := makePair("Cluster", "cluster-ok")
	clusterNok := makePair("Cluster", "cluster-nok")
	namespaceOk := makePair("Namespace", "ns-ok")
	namespaceNok := makePair("Namespace", "ns-nok")

	resolvedRoles := []permissions.ResolvedRole{
		roletest.NewResolvedRole(
			"role1",
			map[string]storage.Access{
				"Namespaces": storage.Access_READ_ACCESS,
			},
			&storage.SimpleAccessScope{
				Rules: &storage.SimpleAccessScope_Rules{
					IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
						{
							ClusterName:   clusterOk.GetValue(),
							NamespaceName: namespaceOk.GetValue(),
						},
					},
				},
			}),
	}

	ctrl := gomock.NewController(t)
	id := mocks.NewMockIdentity(ctrl)
	id.EXPECT().Roles().Return(resolvedRoles).AnyTimes()
	ctx := authn.ContextWithIdentity(context.Background(), id, t)
	testFamily := &dto.MetricFamily{
		Metric: []*dto.Metric{
			{Label: []*dto.LabelPair{clusterOk, namespaceOk}},
			{Label: []*dto.LabelPair{clusterOk, namespaceNok}},
			{Label: []*dto.LabelPair{clusterNok, namespaceOk}},
			{Label: []*dto.LabelPair{clusterNok, namespaceNok}},
		},
	}

	ctx = sac.WithGlobalAccessScopeChecker(ctx,
		sac.TestScopeCheckerCoreFromFullScopeMap(t,
			sac.TestScopeMap{
				storage.Access_READ_ACCESS: {
					resources.Cluster.GetResource(): &sac.TestResourceScope{
						Clusters: map[string]*sac.TestClusterScope{
							clusterOk.GetValue(): {Namespaces: []string{namespaceOk.GetValue(), namespaceNok.GetValue()}},
						},
					},
					resources.Namespace.GetResource(): &sac.TestResourceScope{
						Clusters: map[string]*sac.TestClusterScope{
							clusterOk.GetValue(): {Namespaces: []string{namespaceOk.GetValue()}},
						},
					},
				},
			}))

	gatherer, err := MakeSacGatherer(ctx, &testGatherer{
		testFamily,
	})
	assert.NoError(t, err)

	actual, err := gatherer.Gather()
	assert.NoError(t, err)

	expected := []*dto.MetricFamily{{
		Metric: []*dto.Metric{
			{Label: []*dto.LabelPair{clusterOk, namespaceOk}},
		},
	}}

	assert.Equal(t, expected, actual)
	// test that testFamily is modified in place:
	assert.Equal(t, expected, []*dto.MetricFamily{testFamily})
}
