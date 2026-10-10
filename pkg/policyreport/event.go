// Package policyreport canonicalizes external PolicyReport/ClusterPolicyReport
// results (e.g. from Kyverno) into a source-neutral SecurityEvent envelope.
//
// This package is deliberately free of Kubernetes client, Sensor, and Central
// wiring: adapters here are pure functions from a raw report object to a
// deterministic slice of SecurityEvent values. See security-event-plan.md at
// the repo root for the full design and phased rollout this package is part of.
package policyreport

import "time"

// Source identifies the normalized producer of a SecurityEvent. Kept as a
// bounded string rather than an enum so a standards-based report producer can
// be supported without a new release, per security-event-plan.md's "Source
// normalization" section.
type Source string

// SourceKyverno is the canonical source value for Kyverno-produced reports.
const SourceKyverno Source = "Kyverno"

// EventType identifies which typed payload a SecurityEvent carries.
type EventType string

// EventTypePolicyResult is the only EventType implemented so far.
const EventTypePolicyResult EventType = "PolicyResult"

// PolicyResultOutcome mirrors a PolicyReport result's `result` field.
type PolicyResultOutcome string

// Outcomes a PolicyReport result can report. Only Fail, Error, and Warn are
// actionable in the MVP scope (see ActionableOutcomes) — Pass and Skip are
// explicitly deferred per security-event-plan.md.
const (
	PolicyResultPass  PolicyResultOutcome = "pass"
	PolicyResultFail  PolicyResultOutcome = "fail"
	PolicyResultWarn  PolicyResultOutcome = "warn"
	PolicyResultError PolicyResultOutcome = "error"
	PolicyResultSkip  PolicyResultOutcome = "skip"
)

// ActionableOutcomes are the PolicyResultOutcome values that produce a
// SecurityEvent. Pass and skip results are parsed (so callers can compute
// summaries) but never canonicalized into events.
var ActionableOutcomes = map[PolicyResultOutcome]bool{
	PolicyResultFail:  true,
	PolicyResultError: true,
	PolicyResultWarn:  true,
}

// Severity mirrors a result's optional `severity` field. Producers are not
// required to set one.
type Severity string

// Severity values a producer may report. SeverityUnknown means the producer
// didn't set one at all — this is common; severity is opt-in per policy.
const (
	SeverityUnknown  Severity = ""
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// EntityType identifies what kind of ACS-resolvable entity a SecurityEvent's
// ResolvedEntity refers to. Only Deployment is implemented so far; Node is
// listed in security-event-plan.md as a later expansion.
type EntityType string

// EntityTypeUnknown means resolution hasn't happened yet (or failed). This
// package never sets ResolvedEntity — that's Phase 3's Pod-to-Deployment
// resolver, layered on top of this pure canonicalizer.
const (
	EntityTypeUnknown    EntityType = ""
	EntityTypeDeployment EntityType = "Deployment"
)

// ReportRef identifies the producer report object a SecurityEvent was
// extracted from (a single PolicyReport or ClusterPolicyReport object).
type ReportRef struct {
	APIVersion      string
	Kind            string // "PolicyReport" or "ClusterPolicyReport"
	UID             string
	Name            string
	Namespace       string // empty for ClusterPolicyReport
	ResourceVersion string
}

// Subject identifies the original Kubernetes object the report result
// concerns. For Kyverno this comes from the report's top-level `scope` field
// — one report object exists per subject resource, not one report per
// namespace with a results-array of resources (confirmed against a real
// Kyverno v1.18.2 cluster; see pkg/policyreport/testdata/raw/README.md).
type Subject struct {
	APIVersion string
	Kind       string
	UID        string
	Name       string
	Namespace  string
}

// ResolvedEntity is the ACS entity a Subject was resolved to. Always the zero
// value coming out of this package — populated by a later resolution stage.
type ResolvedEntity struct {
	Type EntityType
	ID   string
}

// PolicyResult is the typed payload for EventTypePolicyResult events.
type PolicyResult struct {
	ReportedPolicy   string
	ReportedRule     string
	Category         string
	Result           PolicyResultOutcome
	ReportedSeverity Severity
	Message          string
	// OriginalSource preserves the report result's own `source` field
	// verbatim, for diagnostics — never used for matching (see Source).
	OriginalSource string
	// Properties preserves unknown/producer-specific key-value pairs
	// without promoting them to policy criteria (security-event-plan.md:
	// "retain them for details and diagnostics").
	Properties map[string]string
}

// SecurityEvent is the canonical, source-neutral ingestion envelope described
// in security-event-plan.md's "Canonical domain contract".
type SecurityEvent struct {
	// ID is a stable identity computed by ComputeID. Two SecurityEvents
	// with the same ID represent the same logical finding across updates.
	ID         string
	Source     Source
	Type       EventType
	ObservedAt time.Time

	Report         ReportRef
	Subject        Subject
	ResolvedEntity ResolvedEntity

	Details PolicyResult
}
