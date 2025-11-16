package storagewrappers

import (
	"github.com/stackrox/rox/generated/storage"
)

type AdvisoryWrapper struct {
	*storage.Advisory
}

func (w *AdvisoryWrapper) AsAdvisory() *storage.Advisory {
	if w == nil {
		return nil
	}
	return w.Advisory
}

func (w *AdvisoryWrapper) SetName(name string) {
	if w == nil || w.Advisory == nil {
		return
	}
	w.Name = name
}

func (w *AdvisoryWrapper) SetLink(link string) {
	if w == nil || w.Advisory == nil {
		return
	}
	w.Link = link
}
