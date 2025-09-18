package policies

import (
	"strconv"

	"github.com/stackrox/rox/central/metrics/custom/tracker"
)

var LazyLabels = []tracker.LazyLabel[*finding]{
	{Label: "Enabled", Getter: func(f *finding) string {
		return strconv.FormatBool(f.enabled)
	}},
}

type finding struct {
	err     error
	enabled bool
	n       int
}

func (f *finding) GetError() error {
	return f.err
}

func (f *finding) GetIncrement() int {
	return f.n
}
