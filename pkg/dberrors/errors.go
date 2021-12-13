package dberrors

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/pkg/errorhelpers"
)

func message(t, id string) string {
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

// New formats a error message and returns a RoxError wrapping a new error.
func New(t, id string) errorhelpers.RoxError {
	return errorhelpers.ErrNotFound.Wraps(message(t, id))
}
