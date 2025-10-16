package transform

import (
	"reflect"

	"github.com/hashicorp/go-multierror"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protocompat"
)

var _ Transformer = (*accessScopeTransform)(nil)

type accessScopeTransform struct{}

func newAccessScopeTransform() *accessScopeTransform {
	return &accessScopeTransform{}
}

func (a *accessScopeTransform) Transform(configuration declarativeconfig.Configuration) (map[reflect.Type][]protocompat.Message, error) {
	scopeConfig, ok := configuration.(*declarativeconfig.AccessScope)
	if !ok {
		return nil, errox.InvalidArgs.Newf("invalid configuration type received for access scope: %T", configuration)
	}
	rules, err := rulesFromScopeConfig(scopeConfig)
	if err != nil {
		return nil, errox.InvalidArgs.CausedBy(err)
	}
	traits := &storage.Traits{}
	traits.SetOrigin(storage.Traits_DECLARATIVE)
	scopeProto := &storage.SimpleAccessScope{}
	scopeProto.SetId(declarativeconfig.NewDeclarativeAccessScopeUUID(scopeConfig.Name).String())
	scopeProto.SetName(scopeConfig.Name)
	scopeProto.SetDescription(scopeConfig.Description)
	scopeProto.SetRules(rules)
	scopeProto.SetTraits(traits)

	return map[reflect.Type][]protocompat.Message{
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

	sr := &storage.SimpleAccessScope_Rules{}
	sr.SetIncludedClusters(includedClustersFromScopeConfig(scope))
	sr.SetIncludedNamespaces(includedNamespacesFromScopeConfig(scope))
	sr.SetClusterLabelSelectors(clusterLabelSelectors)
	sr.SetNamespaceLabelSelectors(namespaceLabelSelectors)
	return sr, nil
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
			srn := &storage.SimpleAccessScope_Rules_Namespace{}
			srn.SetClusterName(obj.Cluster)
			srn.SetNamespaceName(namespace)
			namespaces = append(namespaces, srn)
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
			sr := &storage.SetBasedLabelSelector_Requirement{}
			sr.SetKey(req.Key)
			sr.SetOp(storage.SetBasedLabelSelector_Operator(req.Operator))
			sr.SetValues(req.Values)
			reqs = append(reqs, sr)
		}
		sbls := &storage.SetBasedLabelSelector{}
		sbls.SetRequirements(reqs)
		setBasedLabelSelectors = append(setBasedLabelSelectors, sbls)
	}
	return setBasedLabelSelectors, labelSelectorErrs.ErrorOrNil()
}
