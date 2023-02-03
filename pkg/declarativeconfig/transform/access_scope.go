package transform

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
)

var _ Transformer = (*accessScopeTransform)(nil)

type accessScopeTransform struct{}

func newAccessScopeTransform() *accessScopeTransform {
	return &accessScopeTransform{}
}

func (a *accessScopeTransform) Transform(configuration declarativeconfig.Configuration) ([]proto.Message, error) {
	scopeConfig, ok := configuration.(*declarativeconfig.AccessScope)
	if !ok {
		return nil, errox.InvalidArgs.Newf("invalid configuration type received for access scope: %T", configuration)
	}
	scopeProto := &storage.SimpleAccessScope{
		Id:          declarativeconfig.NewDeclarativeAccessScopeUUID(scopeConfig.Name).String(),
		Name:        scopeConfig.Name,
		Description: scopeConfig.Description,
		Rules:       rulesFromScopeConfig(scopeConfig),
		Traits: &storage.Traits{
			Origin: storage.Traits_DECLARATIVE,
		},
	}
	return []proto.Message{scopeProto}, nil
}

func rulesFromScopeConfig(scope *declarativeconfig.AccessScope) *storage.SimpleAccessScope_Rules {
	return &storage.SimpleAccessScope_Rules{
		IncludedClusters:        includedClustersFromScopeConfig(scope),
		IncludedNamespaces:      includedNamespacesFromScopeConfig(scope),
		ClusterLabelSelectors:   clusterLabelSelectorsFromScopeConfig(scope),
		NamespaceLabelSelectors: namespaceLabelSelectorsFromScopeConfig(scope),
	}
}

func includedClustersFromScopeConfig(scope *declarativeconfig.AccessScope) []string {
	clusters := make([]string, 0, len(scope.Rules.IncludedObjects))
	for _, obj := range scope.Rules.IncludedObjects {
		clusters = append(clusters, obj.Cluster)
	}
	return clusters
}

func includedNamespacesFromScopeConfig(scope *declarativeconfig.AccessScope) []*storage.SimpleAccessScope_Rules_Namespace {
	var namespaces []*storage.SimpleAccessScope_Rules_Namespace
	for _, obj := range scope.Rules.IncludedObjects {
		for _, namespace := range obj.Namespaces {
			namespaces = append(namespaces, &storage.SimpleAccessScope_Rules_Namespace{
				ClusterName:   obj.Cluster,
				NamespaceName: namespace,
			})
		}
	}
	return namespaces
}

func clusterLabelSelectorsFromScopeConfig(scope *declarativeconfig.AccessScope) []*storage.SetBasedLabelSelector {
	var clusterLabelSelectors []*storage.SetBasedLabelSelector
	for _, cl := range scope.Rules.ClusterLabelSelectors {
		reqs := make([]*storage.SetBasedLabelSelector_Requirement, 0, len(cl.Requirements))
		for _, req := range cl.Requirements {
			reqs = append(reqs, &storage.SetBasedLabelSelector_Requirement{
				Key:    req.Key,
				Op:     storage.SetBasedLabelSelector_Operator(req.Operator),
				Values: req.Values,
			})
		}
		clusterLabelSelectors = append(clusterLabelSelectors, &storage.SetBasedLabelSelector{
			Requirements: reqs,
		})
	}
	return clusterLabelSelectors
}

func namespaceLabelSelectorsFromScopeConfig(scope *declarativeconfig.AccessScope) []*storage.SetBasedLabelSelector {
	var namespaceLabelSelector []*storage.SetBasedLabelSelector
	for _, nl := range scope.Rules.NamespaceLabelSelectors {
		reqs := make([]*storage.SetBasedLabelSelector_Requirement, 0, len(nl.Requirements))
		for _, req := range nl.Requirements {
			reqs = append(reqs, &storage.SetBasedLabelSelector_Requirement{
				Key:    req.Key,
				Op:     storage.SetBasedLabelSelector_Operator(req.Operator),
				Values: req.Values,
			})
		}
		namespaceLabelSelector = append(namespaceLabelSelector, &storage.SetBasedLabelSelector{
			Requirements: reqs,
		})
	}
	return namespaceLabelSelector
}
