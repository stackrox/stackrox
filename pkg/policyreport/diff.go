package policyreport

import "sort"

// Diff reconciles a previous snapshot of SecurityEvents against a new
// snapshot for the same report, keyed by SecurityEvent.ID, returning which
// events to create, update, or remove. Per security-event-plan.md's "Treat
// reports as current state" guiding decision:
//
//   - new minus old  -> created
//   - shared identity, changed content -> updated
//   - old minus new  -> removed
//
// Report deletion is modeled by calling Diff(previousEvents, nil) — every
// previously-actionable event is returned in removed.
//
// All three return slices are sorted by ID for determinism.
func Diff(oldEvents, newEvents []SecurityEvent) (created, updated, removed []SecurityEvent) {
	oldByID := indexByID(oldEvents)
	newByID := indexByID(newEvents)

	for id, newEvent := range newByID {
		oldEvent, existed := oldByID[id]
		switch {
		case !existed:
			created = append(created, newEvent)
		case !equalContent(oldEvent, newEvent):
			updated = append(updated, newEvent)
		}
	}
	for id, oldEvent := range oldByID {
		if _, stillPresent := newByID[id]; !stillPresent {
			removed = append(removed, oldEvent)
		}
	}

	sortByID(created)
	sortByID(updated)
	sortByID(removed)
	return created, updated, removed
}

func indexByID(events []SecurityEvent) map[string]SecurityEvent {
	byID := make(map[string]SecurityEvent, len(events))
	for _, e := range events {
		byID[e.ID] = e
	}
	return byID
}

func sortByID(events []SecurityEvent) {
	sort.Slice(events, func(i, j int) bool { return events[i].ID < events[j].ID })
}

// equalContent reports whether two SecurityEvents sharing the same ID
// otherwise carry identical content. ObservedAt is derived from the report's
// own result timestamp (not wall-clock at parse time), so re-canonicalizing
// unchanged report content is expected to produce byte-identical events —
// this is what makes repeated processing idempotent (see
// kyverno_v1alpha2_test.go).
func equalContent(a, b SecurityEvent) bool {
	if a.Source != b.Source || a.Type != b.Type || !a.ObservedAt.Equal(b.ObservedAt) {
		return false
	}
	if a.Report != b.Report || a.Subject != b.Subject || a.ResolvedEntity != b.ResolvedEntity {
		return false
	}
	if a.Details.ReportedPolicy != b.Details.ReportedPolicy ||
		a.Details.ReportedRule != b.Details.ReportedRule ||
		a.Details.Category != b.Details.Category ||
		a.Details.Result != b.Details.Result ||
		a.Details.ReportedSeverity != b.Details.ReportedSeverity ||
		a.Details.Message != b.Details.Message ||
		a.Details.OriginalSource != b.Details.OriginalSource {
		return false
	}
	return stringMapsEqual(a.Details.Properties, b.Details.Properties)
}

func stringMapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}
	return true
}
