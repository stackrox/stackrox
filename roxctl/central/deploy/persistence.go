package deploy

import (
	"fmt"

	"github.com/stackrox/rox/roxctl/central/deploy/renderer"
)

type persistenceTypeWrapper struct {
	PersistenceType *renderer.PersistenceType
}

func (f *persistenceTypeWrapper) String() string {
	return f.PersistenceType.String()
}

func (f *persistenceTypeWrapper) Set(input string) error {
	pt, ok := renderer.StringToPersistentTypes[input]
	if !ok {
		return fmt.Errorf("Invalid persistence type: %s", input)
	}
	*f.PersistenceType = pt
	return nil
}

func (f *persistenceTypeWrapper) Type() string {
	return "persistence type"
}
