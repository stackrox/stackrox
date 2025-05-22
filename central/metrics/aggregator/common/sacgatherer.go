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

// Check the metrics labels with SAC.
func (g *SacGatherer) Check(m *dto.Metric) bool {
	for _, label := range m.GetLabel() {
		switch label.GetName() {
		case "Cluster":
			eas, err := g.EffectiveAccessScope(
				permissions.View(resources.Cluster))
			if err != nil {
				return false
			}
			clusterAllowed := false
			for clusterName := range eas.Clusters {
				if clusterName == label.GetValue() {
					clusterAllowed = true
					break
				}
			}
			if !clusterAllowed {
				log.Info("SAC check failed for cluster ", label.GetValue())
				return false
			}
		case "Namespace":
			eas, err := g.EffectiveAccessScope(
				permissions.View(resources.Namespace))
			if err != nil {
				return false
			}
			nsAllowed := true
		clusters:
			for _, cluster := range eas.Clusters {
				switch cluster.State {
				case effectiveaccessscope.Included:
					nsAllowed = true
					continue
				case effectiveaccessscope.Excluded:
					nsAllowed = false
					break clusters
				case effectiveaccessscope.Partial:
					nsAllowed = false
					for nsName := range cluster.Namespaces {
						if nsName == label.GetValue() {
							nsAllowed = true
							break clusters
						}
					}
				}
			}
			if !nsAllowed {
				log.Info("SAC check failed for namespace ", label.GetValue())
				return false
			}
		}
		log.Info("SAC check passed for ", label.GetName())
	}
	return true
}
