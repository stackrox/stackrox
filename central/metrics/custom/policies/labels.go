package policies

import (
	"strconv"

	"github.com/stackrox/rox/central/metrics/custom/tracker"
)

var LazyLabels = tracker.LazyLabelGetters[*finding]{
	"Enabled": func(f *finding) string {
		return strconv.FormatBool(f.enabled)
	},
}

type finding struct {
	enabled bool
	n       int
}

func (f *finding) GetIncrement() int {
	return f.n
}

var _ tracker.WithIncrement = (*finding)(nil)
