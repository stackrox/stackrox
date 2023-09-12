package writer

import (
	"fmt"
	"reflect"

	"github.com/stackrox/rox/pkg/administration/events"
)

type eqEventsMatcher struct {
	event           *events.AdministrationEvent
	num_occurrences int64
}

func (e eqEventsMatcher) Matches(x interface{}) bool {
	// Assumes that x is of type []*storage.AdministrationEvent with one element.
	cmpPtr := reflect.ValueOf(x).Slice(0, 1).Index(0)
	cmpValue := reflect.Indirect(cmpPtr)
	if e.event.GetDomain() != cmpValue.FieldByName("Domain").String() {
		return false
	}
	if e.event.GetHint() != cmpValue.FieldByName("Hint").String() {
		return false
	}
	if e.event.GetMessage() != cmpValue.FieldByName("Message").String() {
		return false
	}
	if e.event.GetResourceId() != cmpValue.FieldByName("ResourceId").String() {
		return false
	}
	if e.event.GetResourceType() != cmpValue.FieldByName("ResourceType").String() {
		return false
	}
	if e.num_occurrences != cmpValue.FieldByName("NumOccurrences").Int() {
		return false
	}
	return true
}

func (e eqEventsMatcher) String() string {
	return fmt.Sprintf("event %+v with %d occurrences", e.event, e.num_occurrences)
}
