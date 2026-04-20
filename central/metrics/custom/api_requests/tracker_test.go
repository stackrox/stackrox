package api_requests

import (
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/metrics/custom/tracker"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clientprofile"
	"github.com/stackrox/rox/pkg/eventual"
	"github.com/stackrox/rox/pkg/glob"
	"github.com/stackrox/rox/pkg/grpc/common/requestinterceptor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchingProfileHeader(t *testing.T) {
	cases := map[string]struct {
		label          tracker.Label
		profile        clientprofile.RuleSet
		expectedHeader glob.Pattern
		expectedValue  glob.Pattern
	}{
		"concrete header match": {
			label: "XCustomHeader",
			profile: clientprofile.RuleSet{
				clientprofile.HeaderPattern("X-Custom-Header", "*"),
			},
			expectedHeader: "X-Custom-Header",
			expectedValue:  "*",
		},
		"glob header match": {
			label: "RhServicenowInstance",
			profile: clientprofile.RuleSet{{
				Headers: clientprofile.GlobMap{"Rh-*": "*"},
			}},
			expectedHeader: "Rh-*",
			expectedValue:  "*",
		},
		"no match": {
			label: "CompletelyUnknown",
			profile: clientprofile.RuleSet{{
				Headers: clientprofile.GlobMap{"Rh-*": "*"},
			}},
		},
		"no headers in profile": {
			label: "Anything",
			profile: clientprofile.RuleSet{
				clientprofile.PathPattern("/api/*"),
			},
		},
		"matches across rules": {
			label: "XExact",
			profile: clientprofile.RuleSet{
				{Headers: clientprofile.GlobMap{"Rh-*": "*"}},
				{Headers: clientprofile.GlobMap{"X-Exact": "*"}},
			},
			expectedHeader: "X-Exact",
			expectedValue:  "*",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			header, value := matchingProfileHeader(tc.label, tc.profile)
			assert.Equal(t, tc.expectedHeader, header)
			assert.Equal(t, tc.expectedValue, value)
		})
	}
}

func TestProfileTrackersAcceptExpectedLabels(t *testing.T) {
	// Verify that each profile tracker accepts a configuration with all
	// its derived labels. Descriptor keys must start with the profile name
	// (the descriptor prefix).
	expected := map[string][]string{
		"servicenow": {"Method", "Path", "RhServicenowInstance", "Status", "UserAgent", "UserID"},
		"splunk_ta":  {"Method", "Path", "Status", "UserID"},
		"roxctl":     {"Method", "Path", "RhRoxctlCommand", "RhRoxctlCommandIndex", "Status", "UserAgent", "UserID"},
		"sensor":     {"Method", "Path", "Status", "UserAgent", "UserID"},
		"unknown":    {"Method", "Path", "Status", "UserAgent", "UserID"},
	}

	trackers := map[string]tracker.Tracker{}
	for tr := range Trackers() {
		trackers[tr.Tracker.(*profileTracker).profileName] = tr.Tracker
	}
	for name, labels := range expected {
		t.Run(name, func(t *testing.T) {
			tr, ok := trackers[name]
			require.True(t, ok, "tracker %q not found", name)

			cfg, err := tr.NewConfiguration(&storage.PrometheusMetrics_Group{
				Enabled: true,
				Descriptors: map[string]*storage.PrometheusMetrics_Group_Labels{
					name + "_total": {Labels: labels},
				},
			})
			require.NoError(t, err)
			assert.NotNil(t, cfg)
		})
	}
}

func TestLabelPatternAcceptsDynamicLabels(t *testing.T) {
	profile := clientprofile.RuleSet{{
		Headers: clientprofile.GlobMap{
			"Rh-*":       "*",
			"User-Agent": "*",
		},
	}}

	pt := &profileTracker{
		TrackerBase: tracker.MakeGlobalTrackerBase("test", "test", maps.Clone(commonLabels), nil),
		profileName: "profile",
		profile:     profile,
	}
	pt.KnownLabels = func(descriptors map[string]*storage.PrometheusMetrics_Group_Labels) []tracker.Label {
		labels := commonLabels.Labels()
		for _, desc := range descriptors {
			for _, label := range desc.GetLabels() {
				if hp, _ := matchingProfileHeader(tracker.Label(label), pt.profile); hp != "" {
					labels = append(labels, tracker.Label(label))
				}
			}
		}
		return labels
	}

	// Both concrete (UserAgent) and dynamic (RhServicenowInstance, RhCustomThing)
	// labels should be accepted.
	cfg, err := pt.NewConfiguration(&storage.PrometheusMetrics_Group{
		Enabled: true,
		Descriptors: map[string]*storage.PrometheusMetrics_Group_Labels{
			"profile_total": {Labels: []string{"UserID", "UserAgent", "RhServicenowInstance", "RhCustomThing"}},
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// A label that doesn't match any pattern should still be rejected.
	_, err = pt.NewConfiguration(&storage.PrometheusMetrics_Group{
		Enabled: true,
		Descriptors: map[string]*storage.PrometheusMetrics_Group_Labels{
			"profile_total": {Labels: []string{"CompletelyUnknown"}},
		},
	})
	assert.Error(t, err)
}

func TestFilterDescriptors(t *testing.T) {
	cfg := &storage.PrometheusMetrics_Group{
		Enabled: true,
		Descriptors: map[string]*storage.PrometheusMetrics_Group_Labels{
			"alpha_total": {Labels: []string{"UserID"}},
			"alpha_by_ns": {Labels: []string{"UserID"}},
			"beta_total":  {Labels: []string{"UserID"}},
			"unrelated":   {Labels: []string{"UserID"}},
		},
	}

	filtered := filterMetrics(cfg, "alpha")
	assert.Len(t, filtered.GetDescriptors(), 2)
	assert.Contains(t, filtered.GetDescriptors(), "alpha_total")
	assert.Contains(t, filtered.GetDescriptors(), "alpha_by_ns")

	// No prefix returns original.
	assert.Same(t, cfg, filterMetrics(cfg, ""))

	// No matches returns empty descriptors.
	filtered = filterMetrics(cfg, "gamma")
	assert.Empty(t, filtered.GetDescriptors())
}

func TestAllProfilesEmitMetrics(t *testing.T) {
	// Simulate the runner flow: collect Trackers(), validate and apply a
	// shared configuration, fire requests matching different profiles, and
	// verify that each profile produces its own counter metric.
	metrics.DeleteGlobalRegistry()
	interceptor = eventual.New[*requestinterceptor.RequestInterceptor]()
	SetInterceptor(requestinterceptor.NewRequestInterceptor())

	sharedGroup := &storage.PrometheusMetrics_Group{
		Enabled: true,
		Descriptors: map[string]*storage.PrometheusMetrics_Group_Labels{
			"roxctl":     {Labels: []string{"Status"}},
			"servicenow": {Labels: []string{"Status"}},
			"unknown":    {Labels: []string{"Status"}},
		},
	}

	// Collect trackers the same way runner.go does.
	collected := slices.Collect(Trackers())
	require.Len(t, collected, len(profileTrackers))

	// Validate and apply configuration for each collected tracker.
	for _, reg := range collected {
		cfg, err := reg.Tracker.NewConfiguration(reg.GetGroupConfig(
			&storage.PrometheusMetrics{ApiRequests: sharedGroup}))
		require.NoError(t, err)
		reg.Tracker.Reconfigure(cfg)
	}
	t.Cleanup(func() {
		for _, reg := range collected {
			reg.Tracker.Reconfigure(nil)
		}
		interceptor = eventual.New[*requestinterceptor.RequestInterceptor]()
	})

	RegisterHandler()

	// Simulate requests matching each profile.
	requests := map[string]*requestinterceptor.RequestParams{
		"roxctl": {
			Method:  "GET",
			Path:    "/v1/metadata",
			Code:    200,
			Headers: http.Header{"User-Agent": {"roxctl/4.0"}},
		},
		"servicenow": {
			Method:  "GET",
			Path:    "/v1/alerts",
			Code:    200,
			Headers: http.Header{"User-Agent": {"Mozilla ServiceNow Bot"}},
		},
		"unknown": {
			Method:  "GET",
			Path:    "/v1/clusters",
			Code:    200,
			Headers: http.Header{"User-Agent": {"curl/8.0"}},
		},
	}
	for _, rp := range requests {
		recordRequest(rp)
	}

	globalRegistry, err := metrics.GetGlobalRegistry()
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	globalRegistry.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rec.Body.String()

	for profile := range requests {
		assert.Contains(t, body, "rox_central_api_request_"+profile+"{",
			"expected metric for profile %q", profile)
	}
}

func TestSharedConfigAcceptedByAllTrackers(t *testing.T) {
	// A single config group with descriptors for multiple profiles should be
	// accepted by every tracker — each picks up only its own descriptors.
	sharedConfig := &storage.PrometheusMetrics_Group{
		Enabled: true,
		Descriptors: map[string]*storage.PrometheusMetrics_Group_Labels{
			"servicenow": {Labels: []string{"UserID", "Status", "RhServicenowInstance"}},
			"roxctl":     {Labels: []string{"UserID", "Status", "UserAgent", "RhRoxctlCommand"}},
			"unknown":    {Labels: []string{"UserID", "Status", "UserAgent"}},
		},
	}

	for tr := range Trackers() {
		t.Run(tr.Tracker.(*profileTracker).profileName, func(t *testing.T) {
			cfg, err := tr.NewConfiguration(sharedConfig)
			require.NoError(t, err)
			assert.NotNil(t, cfg)
		})
	}
}
