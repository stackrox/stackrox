package transform

import (
	"reflect"

	"github.com/gogo/protobuf/proto"
	"github.com/hashicorp/go-multierror"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
)

var _ Transformer = (*accessScopeTransform)(nil)

type accessScopeTransform struct{}

func newAccessScopeTransform() *accessScopeTransform {
	return &accessScopeTransform{}
}

func (a *accessScopeTransform) Transform(configuration declarativeconfig.Configuration) (map[reflect.Type][]proto.Message, error) {
	scopeConfig, ok := configuration.(*declarativeconfig.AccessScope)
	if !ok {
		return nil, errox.InvalidArgs.Newf("invalid configuration type received for access scope: %T", configuration)
	}
	rules, err := rulesFromScopeConfig(scopeConfig)
	if err != nil {
		return nil, errox.InvalidArgs.CausedBy(err)
	}
	scopeProto := &storage.SimpleAccessScope{
		Id:          declarativeconfig.NewDeclarativeAccessScopeUUID(scopeConfig.Name).String(),
		Name:        scopeConfig.Name,
		Description: scopeConfig.Description,
		Rules:       rules,
		Traits: &storage.Traits{
			Origin: storage.Traits_DECLARATIVE,
		},
	}

	return map[reflect.Type][]proto.Message{
		reflect.TypeOf((*storage.SimpleAccessScope)(nil)): {scopeProto},
	}, nil
}

func rulesFromScopeConfig(scope *declarativeconfig.AccessScope) (*storage.SimpleAccessScope_Rules, error) {
	clusterLabelSelectors, err := labelSelectorsFromScopeConfig(scope.Rules.ClusterLabelSelectors)
	if err != nil {
		return nil, err
	}
	namespaceLabelSelectors, err := labelSelectorsFromScopeConfig(scope.Rules.NamespaceLabelSelectors)
	if err != nil {
		return nil, err
	}

	return &storage.SimpleAccessScope_Rules{
		IncludedClusters:        includedClustersFromScopeConfig(scope),
		IncludedNamespaces:      includedNamespacesFromScopeConfig(scope),
		ClusterLabelSelectors:   clusterLabelSelectors,
		NamespaceLabelSelectors: namespaceLabelSelectors,
	}, nil
}

func includedClustersFromScopeConfig(scope *declarativeconfig.AccessScope) []string {
	clusters := make([]string, 0, len(scope.Rules.IncludedObjects))
	for _, obj := range scope.Rules.IncludedObjects {
		// An access scope should specify included clusters when there is no namespace set for them.
		// If a namespace is set, the key value pair of cluster/namespace should be a part of the included namespaces
		// instead.
		if len(obj.Namespaces) == 0 {
			clusters = append(clusters, obj.Cluster)
		}
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

func labelSelectorsFromScopeConfig(labelSelectors []declarativeconfig.LabelSelector) ([]*storage.SetBasedLabelSelector, error) {
	var setBasedLabelSelectors []*storage.SetBasedLabelSelector
	var labelSelectorErrs *multierror.Error
	for _, ls := range labelSelectors {
		reqs := make([]*storage.SetBasedLabelSelector_Requirement, 0, len(ls.Requirements))
		for _, req := range ls.Requirements {
			reqs = append(reqs, &storage.SetBasedLabelSelector_Requirement{
				Key:    req.Key,
				Op:     storage.SetBasedLabelSelector_Operator(req.Operator),
				Values: req.Values,
			})
		}
		setBasedLabelSelectors = append(setBasedLabelSelectors, &storage.SetBasedLabelSelector{
			Requirements: reqs,
		})
	}
	return setBasedLabelSelectors, labelSelectorErrs.ErrorOrNil()
}
