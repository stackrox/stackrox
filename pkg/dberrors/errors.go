package dberrors

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
)

func format(t, id string) string {
	sb := strings.Builder{}
	if t != "" {
		_, _ = sb.WriteString(fmt.Sprintf("%s ", t))
	}
	if id != "" {
		_, _ = sb.WriteString(fmt.Sprintf("'%s' ", id))
	}
	_, _ = sb.WriteString("not found")
	return sb.String()
}

var (
	// ErrNotFound is a package specific sentinel error, indicating that the requested object was not found.
	ErrNotFound = errox.New(errox.CodeNotFound, "db", "object not found")
)

// NewNotFound returns a new error associated with the provided type and id.
func NewNotFound(t, id string) error {
	return errors.Wrap(ErrNotFound, format(t, id))
}
