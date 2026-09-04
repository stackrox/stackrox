package policyreport

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// KyvernoV1Alpha2APIVersion is the only report API version this adapter
// understands. Confirmed against a real Kyverno v1.18.2 cluster — see
// pkg/policyreport/testdata/raw/README.md. A future openreports.io/v1alpha1
// (or newer wgpolicyk8s.io) adapter belongs in its own file behind the same
// signature, per security-event-plan.md's "Prefer adapters over a
// version-specific domain model".
const KyvernoV1Alpha2APIVersion = "wgpolicyk8s.io/v1alpha2"

const (
	kindPolicyReport        = "PolicyReport"
	kindClusterPolicyReport = "ClusterPolicyReport"
)

// CanonicalizeKyvernoV1Alpha2 is a pure adapter from a single Kyverno
// wgpolicyk8s.io/v1alpha2 PolicyReport or ClusterPolicyReport object to a
// deterministic slice of canonical SecurityEvents.
//
// Only actionable results (fail, error, warn — see ActionableOutcomes) become
// SecurityEvents; pass/skip results are parsed but produce no event, per the
// plan's MVP scope. A report with no actionable results yields an empty,
// non-nil slice and no error — that's a normal, common case, not a failure.
//
// A malformed individual result (missing policy/rule/result) is skipped
// without discarding its valid siblings. Only a structurally unusable report
// (wrong API version/kind, or a missing/unidentifiable scope) returns an
// error, since without a subject there is nothing to attribute any result to.
func CanonicalizeKyvernoV1Alpha2(clusterID string, report *unstructured.Unstructured) ([]SecurityEvent, error) {
	if report == nil {
		return nil, errors.New("policyreport: report is nil")
	}

	if apiVersion := report.GetAPIVersion(); apiVersion != KyvernoV1Alpha2APIVersion {
		return nil, errors.Errorf("policyreport: unsupported apiVersion %q, expected %q", apiVersion, KyvernoV1Alpha2APIVersion)
	}
	kind := report.GetKind()
	if kind != kindPolicyReport && kind != kindClusterPolicyReport {
		return nil, errors.Errorf("policyreport: unsupported kind %q", kind)
	}

	subject, err := extractSubject(report)
	if err != nil {
		return nil, errors.Wrap(err, "policyreport: extracting subject scope")
	}

	reportRef := ReportRef{
		APIVersion:      report.GetAPIVersion(),
		Kind:            kind,
		UID:             string(report.GetUID()),
		Name:            report.GetName(),
		Namespace:       report.GetNamespace(),
		ResourceVersion: report.GetResourceVersion(),
	}

	rawResults, found, err := unstructured.NestedSlice(report.Object, "results")
	if err != nil {
		return nil, errors.Wrap(err, "policyreport: reading results")
	}
	if !found || len(rawResults) == 0 {
		return []SecurityEvent{}, nil
	}

	events := make([]SecurityEvent, 0, len(rawResults))
	for _, raw := range rawResults {
		resultMap, ok := raw.(map[string]interface{})
		if !ok {
			// A single malformed entry in the results array must not
			// discard its valid siblings.
			continue
		}

		event, ok := canonicalizeResult(clusterID, reportRef, subject, resultMap)
		if !ok {
			continue
		}
		events = append(events, event)
	}

	sort.Slice(events, func(i, j int) bool { return events[i].ID < events[j].ID })
	return events, nil
}

// extractSubject reads the report's top-level `scope` field. Kyverno emits
// exactly one report object per subject resource (see testdata README), so
// unlike the plan's original draft there is no per-result resources array to
// walk — the whole report shares one subject.
func extractSubject(report *unstructured.Unstructured) (Subject, error) {
	scopeMap, found, err := unstructured.NestedMap(report.Object, "scope")
	if err != nil {
		return Subject{}, errors.Wrap(err, "reading scope")
	}
	if !found {
		return Subject{}, errors.New("scope is missing")
	}

	uid := strings.TrimSpace(nestedStringOrEmpty(scopeMap, "uid"))
	if uid == "" {
		return Subject{}, errors.New("scope.uid is missing or empty")
	}

	return Subject{
		APIVersion: sanitizeProducerString(nestedStringOrEmpty(scopeMap, "apiVersion")),
		Kind:       sanitizeProducerString(nestedStringOrEmpty(scopeMap, "kind")),
		UID:        uid,
		Name:       sanitizeProducerString(nestedStringOrEmpty(scopeMap, "name")),
		Namespace:  sanitizeProducerString(nestedStringOrEmpty(scopeMap, "namespace")),
	}, nil
}

// canonicalizeResult converts a single `results[]` entry into a SecurityEvent.
// Returns ok=false when the result is either malformed (missing a required
// field) or simply non-actionable (pass/skip) — both are normal, silent
// skips, not errors.
func canonicalizeResult(clusterID string, reportRef ReportRef, subject Subject, resultMap map[string]interface{}) (SecurityEvent, bool) {
	policyName := strings.TrimSpace(nestedStringOrEmpty(resultMap, "policy"))
	ruleName := strings.TrimSpace(nestedStringOrEmpty(resultMap, "rule"))
	rawOutcome := strings.ToLower(strings.TrimSpace(nestedStringOrEmpty(resultMap, "result")))
	if policyName == "" || ruleName == "" || rawOutcome == "" {
		return SecurityEvent{}, false
	}

	outcome := PolicyResultOutcome(rawOutcome)
	if !isKnownOutcome(outcome) || !ActionableOutcomes[outcome] {
		return SecurityEvent{}, false
	}

	reportedSource := nestedStringOrEmpty(resultMap, "source")
	canonicalSource := normalizeSource(reportedSource)

	details := PolicyResult{
		ReportedPolicy:   sanitizeProducerString(policyName),
		ReportedRule:     sanitizeProducerString(ruleName),
		Category:         sanitizeProducerString(nestedStringOrEmpty(resultMap, "category")),
		Result:           outcome,
		ReportedSeverity: normalizeSeverity(nestedStringOrEmpty(resultMap, "severity")),
		Message:          sanitizeProducerString(nestedStringOrEmpty(resultMap, "message")),
		OriginalSource:   sanitizeProducerString(reportedSource),
		Properties:       extractProperties(resultMap),
	}

	id := ComputeID(clusterID, reportRef.UID, canonicalSource, details.ReportedPolicy, details.ReportedRule, subject.UID)

	return SecurityEvent{
		ID:         id,
		Source:     canonicalSource,
		Type:       EventTypePolicyResult,
		ObservedAt: parseResultTimestamp(resultMap),
		Report:     reportRef,
		Subject:    subject,
		Details:    details,
	}, true
}

func isKnownOutcome(o PolicyResultOutcome) bool {
	switch o {
	case PolicyResultPass, PolicyResultFail, PolicyResultWarn, PolicyResultError, PolicyResultSkip:
		return true
	default:
		return false
	}
}

// normalizeSource maps a report result's own `source` string to a canonical
// Source. This adapter is registered specifically for Kyverno-shaped
// wgpolicyk8s.io/v1alpha2 reports, so its own adapter identity (Kyverno) is
// always the fallback, per security-event-plan.md's source-normalization
// precedence ("explicit report result source, then ... adapter identity") —
// the original string is preserved separately in PolicyResult.OriginalSource
// for diagnostics regardless of what this returns.
func normalizeSource(_ string) Source {
	return SourceKyverno
}

func normalizeSeverity(raw string) Severity {
	switch Severity(strings.ToLower(strings.TrimSpace(raw))) {
	case SeverityLow:
		return SeverityLow
	case SeverityMedium:
		return SeverityMedium
	case SeverityHigh:
		return SeverityHigh
	case SeverityCritical:
		return SeverityCritical
	default:
		return SeverityUnknown
	}
}

func extractProperties(resultMap map[string]interface{}) map[string]string {
	rawProps, found, err := unstructured.NestedMap(resultMap, "properties")
	if err != nil || !found || len(rawProps) == 0 {
		return nil
	}
	props := make(map[string]string, len(rawProps))
	for k, v := range rawProps {
		props[sanitizeProducerString(k)] = sanitizeProducerString(fmt.Sprintf("%v", v))
	}
	return props
}

func parseResultTimestamp(resultMap map[string]interface{}) time.Time {
	tsMap, found, err := unstructured.NestedMap(resultMap, "timestamp")
	if err != nil || !found {
		return time.Time{}
	}
	seconds, secondsOK := toInt64(tsMap["seconds"])
	nanos, _ := toInt64(tsMap["nanos"])
	if !secondsOK || seconds == 0 {
		return time.Time{}
	}
	return time.Unix(seconds, nanos).UTC()
}

// toInt64 tolerates every numeric representation a `map[string]interface{}`
// built from JSON/YAML can realistically contain: encoding/json.Unmarshal
// produces float64, the k8s-flavored decoders can produce int64 or
// json.Number depending on path. Being defensive here is what lets
// CanonicalizeKyvernoV1Alpha2 never panic on the unstructured boundary,
// regardless of which decoder produced the object.
func toInt64(v interface{}) (int64, bool) {
	switch t := v.(type) {
	case int64:
		return t, true
	case int:
		return int64(t), true
	case float64:
		return int64(t), true
	case json.Number:
		n, err := t.Int64()
		return n, err == nil
	default:
		return 0, false
	}
}

func nestedStringOrEmpty(m map[string]interface{}, field string) string {
	s, found, err := unstructured.NestedString(m, field)
	if err != nil || !found {
		return ""
	}
	return s
}
