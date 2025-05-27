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
	*effectiveaccessscope.ScopeTree
	prometheus.Gatherer
}

func MakeSacGatherer(ctx context.Context, source prometheus.Gatherer) (prometheus.Gatherer, error) {
	eas, err := sac.GlobalAccessScopeChecker(ctx).
		EffectiveAccessScope(permissions.View(resources.CVE))
	if err != nil {
		return nil, err
	}

	return &SacGatherer{eas, source}, nil
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
	log.Info("checking ", labelClusterName, "/", labelNamespaceName)
	if labelClusterName == "" {
		log.Info("empty cluster name")
		// Do not allow namespace without cluster.
		return labelNamespaceName == ""
	}
	log.Info("eas state", g.State)
	if g.State != effectiveaccessscope.Partial {
		return g.State == effectiveaccessscope.Included
	}
	if len(g.Clusters) == 0 {
		log.Info("no clusters in parital eas")
		return false
	}
	for clusterName, cluster := range g.Clusters {
		log.Info("checking for cluster ", clusterName)
		if clusterName != labelClusterName {
			continue
		}
		log.Info("cluster eas state", cluster.State)
		if cluster.State != effectiveaccessscope.Partial {
			return cluster.State == effectiveaccessscope.Included
		}
		if labelNamespaceName == "" || len(cluster.Namespaces) == 0 {
			log.Info("empty namespace or no cluster namespaces in partial eas")
			return false // false if no namespace? Should be true of no other labels.
		}
		for namespaceName, namespace := range cluster.Namespaces {
			log.Info("checking for namespace ", namespaceName)
			if namespaceName != labelNamespaceName {
				continue
			}
			log.Info("namespace eas", namespace.State)
			return namespace.State != effectiveaccessscope.Excluded // true for partial?
		}
		return false // the namespace is not in the cluster namespaces.
	}
	log.Info("cluster not in the clusters")
	return false
}
