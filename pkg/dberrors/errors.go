package dberrors

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errorhelpers"
)

// NOTE: We might want to introduce a DBNotFound error class to distinguish it
// from generic NotFound errors (using an OverrideMessage() helper).
func NotFound(typ string, ID string) error {
	return errors.Wrapf(errorhelpers.ErrNotFound, "%s '%s'", typ, ID)
}
