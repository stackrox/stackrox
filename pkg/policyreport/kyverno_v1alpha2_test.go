package policyreport

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

const testClusterID = "test-cluster-id"

// loadFixtureList reads a captured `kubectl get policyreport -o yaml` List
// dump (see testdata/raw/README.md for provenance) and returns each item as
// its own *unstructured.Unstructured, matching what a real CRD informer
// hands the adapter one object at a time.
func loadFixtureList(t *testing.T, path string) []*unstructured.Unstructured {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var list struct {
		Items []map[string]interface{} `json:"items"`
	}
	require.NoError(t, yaml.Unmarshal(data, &list))
	require.NotEmpty(t, list.Items, "fixture %s has no items", path)

	reports := make([]*unstructured.Unstructured, 0, len(list.Items))
	for _, item := range list.Items {
		reports = append(reports, &unstructured.Unstructured{Object: item})
	}
	return reports
}

func canonicalizeAll(t *testing.T, reports []*unstructured.Unstructured) []SecurityEvent {
	t.Helper()
	var all []SecurityEvent
	for _, r := range reports {
		events, err := CanonicalizeKyvernoV1Alpha2(testClusterID, r)
		require.NoError(t, err)
		all = append(all, events...)
	}
	return all
}

// TestCanonicalizeKyvernoV1Alpha2_RealFixtures exercises the adapter against
// real `oc get policyreport -o yaml` captures (see testdata/raw/README.md for
// exactly how/where they were produced), not hand-guessed data.
func TestCanonicalizeKyvernoV1Alpha2_RealFixtures(t *testing.T) {
	for name, tc := range map[string]struct {
		fixture             string
		expectedTotalEvents int
	}{
		"new failures and multi-rule failure": {
			fixture:             "testdata/raw/01-new-failures-and-multi-rule.yaml",
			expectedTotalEvents: 9, // 6 report objects; see file for the fail/pass breakdown per object.
		},
		"fail-to-pass via pod replacement": {
			fixture:             "testdata/raw/02-fail-to-pass-via-pod-replacement.yaml",
			expectedTotalEvents: 7,
		},
		"after deployment deletion": {
			fixture:             "testdata/raw/03-after-deployment-deletion.yaml",
			expectedTotalEvents: 1, // only the stale, scaled-to-zero ReplicaSet report still fails.
		},
	} {
		t.Run(name, func(t *testing.T) {
			reports := loadFixtureList(t, tc.fixture)
			events := canonicalizeAll(t, reports)
			assert.Len(t, events, tc.expectedTotalEvents)

			// Every actionable event must carry a non-empty ID, policy, rule,
			// and subject UID — no partially-populated events.
			for _, e := range events {
				assert.NotEmpty(t, e.ID)
				assert.Equal(t, SourceKyverno, e.Source)
				assert.Equal(t, EventTypePolicyResult, e.Type)
				assert.NotEmpty(t, e.Details.ReportedPolicy)
				assert.NotEmpty(t, e.Details.ReportedRule)
				assert.NotEmpty(t, e.Subject.UID)
				assert.True(t, ActionableOutcomes[e.Details.Result], "non-actionable outcome leaked through: %s", e.Details.Result)
			}
		})
	}
}

// TestCanonicalizeKyvernoV1Alpha2_SpotCheckKnownResult pins the exact field
// values for one specific, known result from fixture 1, so a future
// refactor can't silently change field mapping without a test noticing.
func TestCanonicalizeKyvernoV1Alpha2_SpotCheckKnownResult(t *testing.T) {
	reports := loadFixtureList(t, "testdata/raw/01-new-failures-and-multi-rule.yaml")

	var found *SecurityEvent
	for _, r := range reports {
		if r.GetKind() != kindPolicyReport {
			continue
		}
		scope, _, _ := unstructured.NestedMap(r.Object, "scope")
		if nestedStringOrEmpty(scope, "kind") != "Pod" || nestedStringOrEmpty(scope, "name") != "fail-multi-rule-cdb6d9f8d-pdmq4" {
			continue
		}
		events, err := CanonicalizeKyvernoV1Alpha2(testClusterID, r)
		require.NoError(t, err)
		for i := range events {
			if events[i].Details.ReportedRule == "disallow-latest-tag" {
				found = &events[i]
			}
		}
	}

	require.NotNil(t, found, "expected to find the disallow-latest-tag result for fail-multi-rule's pod")
	assert.Equal(t, "require-secure-pod-security-context", found.Details.ReportedPolicy)
	assert.Equal(t, PolicyResultFail, found.Details.Result)
	assert.Equal(t, SeverityHigh, found.Details.ReportedSeverity)
	assert.Equal(t, "kyverno", found.Details.OriginalSource)
	assert.Contains(t, found.Details.Message, "Using 'latest' image tag is not allowed")
	assert.Equal(t, "Pod", found.Subject.Kind)
	assert.Equal(t, "fail-multi-rule-cdb6d9f8d-pdmq4", found.Subject.Name)
	assert.Equal(t, "security-event-test", found.Subject.Namespace)
	assert.False(t, found.ObservedAt.IsZero())
}

func TestCanonicalizeKyvernoV1Alpha2_RejectsWrongAPIVersionOrKind(t *testing.T) {
	for name, obj := range map[string]map[string]interface{}{
		"wrong apiVersion": {
			"apiVersion": "wgpolicyk8s.io/v1beta1",
			"kind":       kindPolicyReport,
			"metadata":   map[string]interface{}{"uid": "u1"},
			"scope":      map[string]interface{}{"uid": "s1", "kind": "Pod"},
		},
		"wrong kind": {
			"apiVersion": KyvernoV1Alpha2APIVersion,
			"kind":       "SomethingElse",
			"metadata":   map[string]interface{}{"uid": "u1"},
			"scope":      map[string]interface{}{"uid": "s1", "kind": "Pod"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := CanonicalizeKyvernoV1Alpha2(testClusterID, &unstructured.Unstructured{Object: obj})
			assert.Error(t, err)
		})
	}
}

func TestCanonicalizeKyvernoV1Alpha2_NilReport(t *testing.T) {
	_, err := CanonicalizeKyvernoV1Alpha2(testClusterID, nil)
	assert.Error(t, err)
}

func TestCanonicalizeKyvernoV1Alpha2_MissingOrEmptyScopeIsAnError(t *testing.T) {
	for name, scope := range map[string]interface{}{
		"scope missing entirely": nil,
		"scope.uid empty":        map[string]interface{}{"uid": "", "kind": "Pod"},
	} {
		t.Run(name, func(t *testing.T) {
			obj := map[string]interface{}{
				"apiVersion": KyvernoV1Alpha2APIVersion,
				"kind":       kindPolicyReport,
				"metadata":   map[string]interface{}{"uid": "report-uid"},
				"results": []interface{}{
					map[string]interface{}{"policy": "p", "rule": "r", "result": "fail"},
				},
			}
			if scope != nil {
				obj["scope"] = scope
			}
			_, err := CanonicalizeKyvernoV1Alpha2(testClusterID, &unstructured.Unstructured{Object: obj})
			assert.Error(t, err)
		})
	}
}

func minimalReport(results []interface{}) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": KyvernoV1Alpha2APIVersion,
		"kind":       kindPolicyReport,
		"metadata": map[string]interface{}{
			"uid":       "report-uid-1",
			"name":      "report-name-1",
			"namespace": "ns1",
		},
		"scope": map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"name":       "pod-1",
			"namespace":  "ns1",
			"uid":        "pod-uid-1",
		},
		"results": results,
	}}
}

func TestCanonicalizeKyvernoV1Alpha2_NoResultsFieldOrEmptyIsNotAnError(t *testing.T) {
	for name, results := range map[string][]interface{}{
		"nil results":   nil,
		"empty results": {},
	} {
		t.Run(name, func(t *testing.T) {
			events, err := CanonicalizeKyvernoV1Alpha2(testClusterID, minimalReport(results))
			require.NoError(t, err)
			assert.Empty(t, events)
		})
	}
}

func TestCanonicalizeKyvernoV1Alpha2_OnlyPassOrSkipResultsProduceNoEvents(t *testing.T) {
	events, err := CanonicalizeKyvernoV1Alpha2(testClusterID, minimalReport([]interface{}{
		map[string]interface{}{"policy": "p1", "rule": "r1", "result": "pass"},
		map[string]interface{}{"policy": "p2", "rule": "r2", "result": "skip"},
	}))
	require.NoError(t, err)
	assert.Empty(t, events)
}

// TestCanonicalizeKyvernoV1Alpha2_MalformedResultDoesNotDiscardValidSiblings
// covers Phase 1's validation requirement directly: a single malformed
// result entry must not prevent its valid siblings in the same report from
// being canonicalized.
func TestCanonicalizeKyvernoV1Alpha2_MalformedResultDoesNotDiscardValidSiblings(t *testing.T) {
	events, err := CanonicalizeKyvernoV1Alpha2(testClusterID, minimalReport([]interface{}{
		map[string]interface{}{"policy": "", "rule": "r1", "result": "fail"},    // missing policy
		map[string]interface{}{"policy": "p2", "rule": "", "result": "fail"},    // missing rule
		map[string]interface{}{"policy": "p3", "rule": "r3", "result": ""},      // missing result
		map[string]interface{}{"policy": "p4", "rule": "r4", "result": "bogus"}, // unknown result value
		"not-even-a-map", // wrong type entirely
		map[string]interface{}{"policy": "p5", "rule": "r5", "result": "fail"}, // the one valid entry
	}))
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "p5", events[0].Details.ReportedPolicy)
	assert.Equal(t, "r5", events[0].Details.ReportedRule)
}

func TestCanonicalizeKyvernoV1Alpha2_UnknownOptionalPropertiesDoNotBreakParsing(t *testing.T) {
	events, err := CanonicalizeKyvernoV1Alpha2(testClusterID, minimalReport([]interface{}{
		map[string]interface{}{
			"policy":            "p1",
			"rule":              "r1",
			"result":            "fail",
			"someFutureField":   "unrecognized-by-this-adapter-version",
			"anotherNewFeature": map[string]interface{}{"nested": true},
		},
	}))
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "p1", events[0].Details.ReportedPolicy)
}

// TestCanonicalizeKyvernoV1Alpha2_ReorderingResultsDoesNotChangeIDs is the
// adapter-level counterpart to identity_test.go's ComputeID guarantee:
// shuffling the results array must produce the same set of IDs.
func TestCanonicalizeKyvernoV1Alpha2_ReorderingResultsDoesNotChangeIDs(t *testing.T) {
	forward := []interface{}{
		map[string]interface{}{"policy": "p1", "rule": "r1", "result": "fail"},
		map[string]interface{}{"policy": "p2", "rule": "r2", "result": "fail"},
		map[string]interface{}{"policy": "p3", "rule": "r3", "result": "fail"},
	}
	reversed := []interface{}{forward[2], forward[1], forward[0]}

	forwardEvents, err := CanonicalizeKyvernoV1Alpha2(testClusterID, minimalReport(forward))
	require.NoError(t, err)
	reversedEvents, err := CanonicalizeKyvernoV1Alpha2(testClusterID, minimalReport(reversed))
	require.NoError(t, err)

	assert.ElementsMatch(t, idsOf(forwardEvents), idsOf(reversedEvents))
	// The adapter itself still returns a deterministic (ID-sorted) order
	// regardless of input order.
	assert.Equal(t, forwardEvents, reversedEvents)
}

// TestCanonicalizeKyvernoV1Alpha2_IsIdempotent covers "repeated processing is
// idempotent": canonicalizing the same report object twice must produce
// byte-identical output, including ObservedAt (derived from the report's own
// timestamp field, not wall-clock time).
func TestCanonicalizeKyvernoV1Alpha2_IsIdempotent(t *testing.T) {
	reports := loadFixtureList(t, "testdata/raw/01-new-failures-and-multi-rule.yaml")
	for i, r := range reports {
		first, err := CanonicalizeKyvernoV1Alpha2(testClusterID, r)
		require.NoError(t, err)
		second, err := CanonicalizeKyvernoV1Alpha2(testClusterID, r)
		require.NoError(t, err)
		assert.Equal(t, first, second, "report index %d was not canonicalized idempotently", i)
	}
}

// TestCanonicalizeKyvernoV1Alpha2_NormalizationIsDeterministic covers
// "normalization is deterministic": varying case/whitespace on the
// producer-supplied source string must always normalize to the same
// canonical Source, while the original is preserved verbatim (trimmed) for
// diagnostics.
func TestCanonicalizeKyvernoV1Alpha2_NormalizationIsDeterministic(t *testing.T) {
	for name, source := range map[string]string{
		"lowercase":         "kyverno",
		"uppercase":         "KYVERNO",
		"mixed case":        "Kyverno",
		"surrounding space": "  kyverno  ",
		"empty":             "",
	} {
		t.Run(name, func(t *testing.T) {
			events, err := CanonicalizeKyvernoV1Alpha2(testClusterID, minimalReport([]interface{}{
				map[string]interface{}{"policy": "p1", "rule": "r1", "result": "fail", "source": source},
			}))
			require.NoError(t, err)
			require.Len(t, events, 1)
			assert.Equal(t, SourceKyverno, events[0].Source)
		})
	}
}

// FuzzCanonicalizeKyvernoV1Alpha2 fuzzes the unstructured boundary: arbitrary
// bytes, decoded as YAML/JSON into a generic map and handed to the adapter,
// must never panic — malformed input should surface as an error (or simply
// no events), never a crash.
func FuzzCanonicalizeKyvernoV1Alpha2(f *testing.F) {
	for _, path := range []string{
		"testdata/raw/01-new-failures-and-multi-rule.yaml",
		"testdata/raw/02-fail-to-pass-via-pod-replacement.yaml",
		"testdata/raw/03-after-deployment-deletion.yaml",
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(data)
	}
	// A handful of adversarial seeds: wrong types, deeply nested garbage,
	// truncated structures.
	f.Add([]byte(`{"apiVersion":"wgpolicyk8s.io/v1alpha2","kind":"PolicyReport","results":"not-a-list"}`))
	f.Add([]byte(`{"apiVersion":"wgpolicyk8s.io/v1alpha2","kind":"PolicyReport","scope":123}`))
	f.Add([]byte(`{"results":[{"policy":1,"rule":[1,2,3],"result":{}}]}`))
	f.Add([]byte(``))
	f.Add([]byte(`null`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var single map[string]interface{}
		if err := yaml.Unmarshal(data, &single); err != nil {
			return
		}
		assert.NotPanics(t, func() {
			_, _ = CanonicalizeKyvernoV1Alpha2(testClusterID, &unstructured.Unstructured{Object: single})
		})

		// Also try treating the input as a List of report items, since
		// that's the shape our real fixtures come in.
		var list struct {
			Items []map[string]interface{} `json:"items"`
		}
		if err := yaml.Unmarshal(data, &list); err != nil {
			return
		}
		for _, item := range list.Items {
			assert.NotPanics(t, func() {
				_, _ = CanonicalizeKyvernoV1Alpha2(testClusterID, &unstructured.Unstructured{Object: item})
			})
		}
	})
}
