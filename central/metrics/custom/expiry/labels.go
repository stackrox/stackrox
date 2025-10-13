package expiry

import (
	"github.com/stackrox/rox/central/metrics/custom/tracker"
)

var LazyLabels = []tracker.LazyLabel[*finding]{
	{Label: "Component", Getter: func(f *finding) string { return f.component }},
}

type finding struct {
	component            string
	hoursUntilExpiration int
}

func (f *finding) GetIncrement() int {
	return f.hoursUntilExpiration
}

var _ tracker.WithIncrement = (*finding)(nil)
