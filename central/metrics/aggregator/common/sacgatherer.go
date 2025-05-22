package common

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/sac/resources"
)

type SacGatherer struct {
	sac.ScopeChecker
	prometheus.Gatherer
}

func MakeSacGatherer(ctx context.Context, source prometheus.Gatherer) (prometheus.Gatherer, error) {
	return &SacGatherer{sac.GlobalAccessScopeChecker(ctx), source}, nil
}

func (g *SacGatherer) Gather() ([]*dto.MetricFamily, error) {
	mfs, err := g.Gatherer.Gather()
	if err != nil {
		return nil, err
	}

	for _, mf := range mfs {
		var filtered []*dto.Metric
		for _, m := range mf.Metric {
			if g.Check(m) {
				filtered = append(filtered, m)
			}
		}
		mf.Metric = filtered
	}
	return mfs, nil
}

var _ prometheus.Gatherer = (*SacGatherer)(nil)

func getClusterNamespace(m *dto.Metric) (clusterName string, namespaceName string) {
	for _, label := range m.GetLabel() {
		switch label.GetName() {
		case "Cluster":
			clusterName = label.GetValue()
		case "Namespace":
			namespaceName = label.GetValue()
		}
	}
	return
}

// Check the metrics labels with SAC.
func (g *SacGatherer) Check(m *dto.Metric) bool {
	labelClusterName, labelNamespaceName := getClusterNamespace(m)
	if labelClusterName == "" {
		// Do not allow namespace without cluster.
		return labelNamespaceName == ""
	}
	eas, err := g.EffectiveAccessScope(
		permissions.View(resources.Cluster))
	if err != nil {
		return false
	}
	switch eas.State {
	case effectiveaccessscope.Included:
		return true
	case effectiveaccessscope.Excluded:
		return false
	case effectiveaccessscope.Partial:
		for clusterName, cluster := range eas.Clusters {
			if clusterName != labelClusterName {
				continue
			}
			switch cluster.State {
			case effectiveaccessscope.Included:
				return true
			case effectiveaccessscope.Excluded:
				return false
			case effectiveaccessscope.Partial:
				if labelNamespaceName == "" {
					return false
				}
				for namespaceName, namespace := range cluster.Namespaces {
					if namespaceName != labelNamespaceName {
						continue
					}
					switch namespace.State {
					case effectiveaccessscope.Included:
						return true
					case effectiveaccessscope.Excluded:
						return false
					case effectiveaccessscope.Partial:
						return true // true for partial?
					}
				}
			}
		}
	}
	return true
}
