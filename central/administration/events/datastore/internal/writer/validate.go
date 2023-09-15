package writer

import (
	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/stringutils"
)

func validateAdministrationEvent(event *events.AdministrationEvent) error {
	if event == nil {
		return errox.InvalidArgs.CausedBy("empty event given")
	}

	if stringutils.AtLeastOneEmpty(event.GetDomain(),
		event.GetMessage(),
		event.GetResourceID(),
		event.GetResourceType(),
		event.GetType().String()) {
		return errox.InvalidArgs.CausedBy("all required fields must be set")
	}

	return nil
}
