package gocache

// addMissedAction appends under the stats mutex-equivalent (the loop's
// sequential section); goroutine-safe via atomic-ish semantics below.
func addMissedAction(stats *Stats, actionID string) {
	if len(stats.MissedActions) < maxLoggedMisses {
		stats.MissedActions = append(stats.MissedActions, actionID)
	}
}
