package expiry

import (
	"github.com/stackrox/rox/central/metrics/custom/tracker"
)

var lazyLabels = []tracker.LazyLabel[*finding]{
	{Label: "Component", Getter: func(f *finding) string { return f.component }},
}

func GetLabels() []string {
	result := make([]string, 0, len(lazyLabels))
	for _, l := range lazyLabels {
		result = append(result, string(l.Label))
	}
	return result
}

type finding struct {
	err                  error
	component            string
	hoursUntilExpiration int
}

func (f *finding) GetError() error {
	return f.err
}

func (f *finding) GetIncrement() int {
	return f.hoursUntilExpiration
}
