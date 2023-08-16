package timeutil

import "time"

// TimeRange frames a time range.
type TimeRange struct {
	From time.Time
	To   time.Time
}

//
// Next and Done methods allow for iterating over a time range by given chunks.
//
// Example:
//	tr := TimeRange{From: from, To: to}
//	for !tr.Done() {
//		chunk := tr.Next(time.Hour)
//		// QueryByTime(chunk.From, chunk.To)
//	}
//

// Next returns a new time range for a period of chunk at max.
func (tr *TimeRange) Next(chunk time.Duration) *TimeRange {
	result := *tr
	tr.From = tr.From.Add(chunk)
	if tr.Done() {
		result.To = tr.To
	} else {
		result.To = tr.From
	}

	return &result
}

// Done tells whether the range start has reached the end.
func (tr *TimeRange) Done() bool {
	return !tr.From.Before(tr.To)
}
