package api_requests

import (
	"iter"
	"maps"
	"strings"

	"github.com/stackrox/rox/central/metrics/custom/tracker"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clientprofile"
)

// profileTracker wraps a TrackerBase with api_requests-specific concerns:
// metrics filtering by profile name and dynamic label resolution via glob
// patterns.
type profileTracker struct {
	*tracker.TrackerBase[*finding]
	profileName string
	profile     clientprofile.RuleSet
}

func (pt *profileTracker) NewConfiguration(cfg *storage.PrometheusMetrics_Group) (*tracker.Configuration, error) {
	return pt.TrackerBase.NewConfiguration(filterMetrics(cfg, pt.profileName))
}

func (pt *profileTracker) Reconfigure(cfg *tracker.Configuration) {
	pt.TrackerBase.ResetGetters(resolveGetters(cfg.GetMetrics(), pt.profile))
	pt.TrackerBase.Reconfigure(cfg)
}

// matchesProfileHeader returns true if the label matches any header name
// pattern in the profile (after hyphen-stripping).
func matchesProfileHeader(label tracker.Label, profile clientprofile.RuleSet) bool {
	for _, rule := range profile {
		for header := range rule.Headers {
			if header.Match(string(label)) {
				return true
			}
		}
	}
	return false
}

// resolveGetters builds a getters map from common labels and resolved header
// patterns for the given metric descriptors.
func resolveGetters(metrics tracker.MetricDescriptors, profile clientprofile.RuleSet) tracker.LazyLabelGetters[*finding] {
	getters := maps.Clone(commonLabels)
	for _, labels := range metrics {
		for _, label := range labels {
			if _, isCommon := commonLabels[label]; isCommon {
				continue
			}
			if matchesProfileHeader(label, profile) {
				getters[label] = makeHeaderGetter(label)
			}
		}
	}
	return getters
}

// filterMetrics returns a copy of the group config containing only
// descriptors whose key starts with the given prefix.
func filterMetrics(cfg *storage.PrometheusMetrics_Group, prefix string) *storage.PrometheusMetrics_Group {
	if prefix == "" {
		return cfg
	}
	filtered := &storage.PrometheusMetrics_Group{
		Enabled:                cfg.GetEnabled(),
		GatheringPeriodMinutes: cfg.GetGatheringPeriodMinutes(),
		Descriptors:            make(map[string]*storage.PrometheusMetrics_Group_Labels),
	}
	for key, labels := range cfg.GetDescriptors() {
		if strings.HasPrefix(key, prefix) {
			filtered.Descriptors[key] = labels
		}
	}
	return filtered
}

// profileTrackers maps profile name to its tracker.
var profileTrackers = func() map[string]*profileTracker {
	allProfiles := make(map[string]clientprofile.RuleSet, len(builtinProfiles)+1)
	maps.Copy(allProfiles, builtinProfiles)
	allProfiles[unknownProfile] = clientprofile.RuleSet{
		clientprofile.HeaderPattern("User-Agent", clientprofile.NoHeaderOrAnyValue),
	}

	trackers := make(map[string]*profileTracker, len(allProfiles))
	for name, profile := range allProfiles {
		pt := &profileTracker{
			TrackerBase: tracker.MakeGlobalTrackerBase(
				"api_request",
				"API requests from "+name,
				maps.Clone(commonLabels),
				nil,
			),
			profileName: name,
			profile:     profile,
		}
		pt.KnownLabels = func(descriptors map[string]*storage.PrometheusMetrics_Group_Labels) []tracker.Label {
			labels := commonLabels.Labels()
			for _, desc := range descriptors {
				for _, label := range desc.GetLabels() {
					if matchesProfileHeader(tracker.Label(label), pt.profile) {
						labels = append(labels, tracker.Label(label))
					}
				}
			}
			return labels
		}
		trackers[name] = pt
	}
	return trackers
}()

// Trackers returns all profile trackers for registration with the runner.
func Trackers() iter.Seq[*tracker.Registration] {
	return func(yield func(*tracker.Registration) bool) {
		for _, pt := range profileTrackers {
			reg := &tracker.Registration{
				Tracker:        pt,
				GetGroupConfig: (*storage.PrometheusMetrics).GetApiRequests,
			}
			if !yield(reg) {
				break
			}
		}
	}
}
