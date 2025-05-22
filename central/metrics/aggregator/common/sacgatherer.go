package common

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac"
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
				return false
			}
		case "Namespace":
			eas, err := g.EffectiveAccessScope(
				permissions.View(resources.Namespace))
			if err != nil {
				return false
			}
			nsAllowed := false
			for _, cluster := range eas.Clusters {
				for ns := range cluster.Namespaces {
					if ns == label.GetValue() {
						nsAllowed = true
						break
					}
				}
			}
			if !nsAllowed {
				return false
			}
		}
	}
	return true
}
