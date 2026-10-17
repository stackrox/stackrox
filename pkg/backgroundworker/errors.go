package backgroundworker

import "errors"

// ErrSkipped signals that a run was intentionally skipped (e.g., advisory
// lock not acquired, feature gate disabled). The library does not count
// skipped runs in runs_total or errors_total metrics.
var ErrSkipped = errors.New("run skipped")
