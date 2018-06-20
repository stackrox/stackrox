package db

import (
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
)

// An ErrNotFound indicates that the desired object could not be located.
type ErrNotFound struct {
	Type string
	ID   string
}

func (e ErrNotFound) Error() string {
	sb := strings.Builder{}
	if e.Type != "" {
		sb.WriteString(fmt.Sprintf("%s ", e.Type))
	}
	if e.ID != "" {
		sb.WriteString(fmt.Sprintf("'%s' ", e.ID))
	}
	sb.WriteString("not found")
	return sb.String()
}

// Status implements the StatusError interface.
func (e ErrNotFound) Status() codes.Code {
	return codes.NotFound
}
