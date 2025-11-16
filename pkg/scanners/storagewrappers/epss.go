package storagewrappers

import (
	"github.com/stackrox/rox/generated/storage"
)

type EPSSWrapper struct {
	*storage.EPSS
}

func (w *EPSSWrapper) AsEPSS() *storage.EPSS {
	if w == nil {
		return nil
	}
	return w.EPSS
}

func (w *EPSSWrapper) SetEPSSProbability(probability float32) {
	if w == nil || w.EPSS == nil {
		return
	}
	w.EpssProbability = probability
}

func (w *EPSSWrapper) SetEPSSPercentile(percentile float32) {
	if w == nil || w.EPSS == nil {
		return
	}
	w.EpssPercentile = percentile
}
