package expiry

import (
	"github.com/stackrox/rox/central/metrics/custom/tracker"
)

var LazyLabels = tracker.LazyLabelGetters[*finding]{
	"Component": func(f *finding) string { return f.component },
	"Name":      func(f *finding) string { return f.name },
}

type finding struct {
	component            string
	name                 string
	hoursUntilExpiration int
}

func (f *finding) GetIncrement() int {
	return f.hoursUntilExpiration
}

var _ tracker.WithIncrement = (*finding)(nil)
