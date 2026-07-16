package administrative_events

import (
	"github.com/stackrox/rox/central/metrics/custom/tracker"
	"github.com/stackrox/rox/generated/storage"
)

var LazyLabels = tracker.LazyLabelGetters[*finding]{
	"Type":         func(f *finding) string { return f.GetType().String() },
	"Level":        func(f *finding) string { return f.GetLevel().String() },
	"Domain":       func(f *finding) string { return f.GetDomain() },
	"ResourceType": func(f *finding) string { return f.GetResource().GetType() },
	"ResourceName": func(f *finding) string { return f.GetResource().GetName() },
}

type finding struct {
	*storage.AdministrationEvent
}

func (f *finding) GetIncrement() int {
	return int(f.GetNumOccurrences())
}

var _ tracker.WithIncrement = (*finding)(nil)
